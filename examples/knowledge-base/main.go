package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"goagent/internal/storage/postgres"
	"goagent/internal/storage/postgres/embedding"
	storage_models "goagent/internal/storage/postgres/models"
	"goagent/internal/storage/postgres/repositories"
	"goagent/internal/storage/postgres/services"

	"gopkg.in/yaml.v3"
)

// SearchResult simple search result (local type for backward compatibility)
type SearchResult struct {
	Content string
	Source  string
	Score   float64
}

// Config configuration structure
type Config struct {
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"database"`

	EmbeddingServiceURL string `yaml:"embedding_service_url"`
	EmbeddingModel      string `yaml:"embedding_model"`

	Knowledge struct {
		ChunkSize    int     `yaml:"chunk_size"`
		ChunkOverlap int     `yaml:"chunk_overlap"`
		TopK         int     `yaml:"top_k"`
		MinScore     float64 `yaml:"min_score"`
	} `yaml:"knowledge"`
}

// Chunk document chunk
type Chunk struct {
	Index int
	Text  string
	Hash  string
}

// KnowledgeBase simplified knowledge base interface
type KnowledgeBase struct {
	config    *Config
	pool      *postgres.Pool
	repo      *repositories.KnowledgeRepository
	embedding *embedding.EmbeddingClient
	retrieval *services.SimpleRetrievalService
}

func main() {
	// Command line arguments
	saveFlag := flag.String("save", "", "Path to document to save")
	chatFlag := flag.Bool("chat", false, "Enable chat mode")
	listFlag := flag.Bool("list", false, "List all documents")
	deleteFlag := flag.String("delete", "", "Document ID to delete")
	tenantFlag := flag.String("tenant", "default", "Tenant ID")
	configFlag := flag.String("config", "config.yaml", "Config file path")

	flag.Parse()

	// Load configuration
	config, err := loadConfig(*configFlag)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create knowledge base
	kb, err := NewKnowledgeBase(config)
	if err != nil {
		log.Fatalf("Failed to create knowledge base: %v", err)
	}
	defer func() {
		if err := kb.Close(); err != nil {
			log.Fatal("Failed to close knowledge base: ", err)
		}
	}()

	ctx := context.Background()

	// Handle different modes
	switch {
	case *saveFlag != "":
		// Import document mode - set 5 minute timeout
		importCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		log.Printf("Importing document: %s", *saveFlag)
		docID, err := kb.ImportDocuments(importCtx, *tenantFlag, *saveFlag)
		cancel()

		if err != nil {
			if err == context.DeadlineExceeded {
				log.Fatalf("Import timeout (5 minutes exceeded)")
			}
			log.Fatalf("Failed to import document: %v", err)
		}
		log.Printf("Document imported successfully. Document ID: %s", docID)

	case *chatFlag:
		// Q&A mode
		log.Println("Chat mode. Enter your questions (type 'exit' to quit):")
		kb.StartChat(ctx, *tenantFlag)

	case *listFlag:
		// List all documents - set 30 second timeout
		listCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		docs, err := kb.ListDocuments(listCtx, *tenantFlag)
		cancel()

		if err != nil {
			if err == context.DeadlineExceeded {
				log.Fatalf("List timeout")
			}
			log.Fatalf("Failed to list documents: %v", err)
		}
		if len(docs) == 0 {
			log.Println("No documents found")
		} else {
			log.Println("Documents:")
			for _, doc := range docs {
				log.Printf("  - ID: %s, Source: %s, Chunks: %d", doc.ID, doc.Source, doc.ChunkCount)
			}
		}

	case *deleteFlag != "":
		// Delete document - set 30 second timeout
		deleteCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		log.Printf("Deleting document: %s", *deleteFlag)
		if err := kb.DeleteDocument(deleteCtx, *tenantFlag, *deleteFlag); err != nil {
			cancel()
			if err == context.DeadlineExceeded {
				log.Fatalf("Delete timeout")
			}
			log.Fatalf("Failed to delete document: %v", err)
		}
		cancel()
		log.Printf("Document deleted successfully")

	default:
		printUsage()
	}
}

// NewKnowledgeBase create knowledge base instance
func NewKnowledgeBase(config *Config) (*KnowledgeBase, error) {
	// Create database configuration
	dbConfig := &postgres.Config{
		Host:            config.Database.Host,
		Port:            config.Database.Port,
		User:            config.Database.User,
		Password:        config.Database.Password,
		Database:        config.Database.Database,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		QueryTimeout:    30 * time.Second,
		Embedding:       postgres.DefaultEmbeddingConfig(),
	}

	// Create database connection pool
	pool, err := postgres.NewPool(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	// Create embedding service client
	embeddingClient := embedding.NewEmbeddingClient(
		config.EmbeddingServiceURL,
		config.EmbeddingModel,
		nil,
		30*time.Second,
	)

	// Create knowledge repository
	kbRepo := repositories.NewKnowledgeRepository(pool.GetDB(), pool.GetDB())

	// Create simple retrieval service (pure vector similarity, no complex weights)
	retrievalService := services.NewSimpleRetrievalService(
		kbRepo,
		embeddingClient,
		&services.SimpleRetrievalConfig{
			TopK:        config.Knowledge.TopK,
			MinScore:    config.Knowledge.MinScore,
			QueryPrefix: "query:",
		},
	)

	return &KnowledgeBase{
		config:    config,
		pool:      pool,
		repo:      kbRepo,
		embedding: embeddingClient,
		retrieval: retrievalService,
	}, nil
}

// ImportDocuments import documents to knowledge base
func (kb *KnowledgeBase) ImportDocuments(ctx context.Context, tenantID, docPath string) (string, error) {
	// Read document content
	content, err := os.ReadFile(docPath)
	if err != nil {
		return "", fmt.Errorf("read document: %w", err)
	}

	// Generate document ID
	docID := uuid.New().String()

	// Chunk processing
	chunks := kb.chunkDocument(string(content), kb.config.Knowledge.ChunkSize, kb.config.Knowledge.ChunkOverlap)
	log.Printf("Document split into %d chunks", len(chunks))

	// Batch process chunks
	successCount := 0
	for i, chunk := range chunks {
		log.Printf("Processing chunk %d/%d... content: %s", i+1, len(chunks), truncateString(chunk.Text, 100))

		// Generate embedding vector (with timeout)
		chunkCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		embedding, err := kb.embedding.EmbedWithPrefix(chunkCtx, chunk.Text, "passage:")
		cancel()

		if err != nil {
			log.Printf("Failed to embed chunk %d: %v (skipping)", i, err)
			continue
		}

		// IMPORTANT: Normalize embedding vector for pgvector cosine distance
		// pgvector's <=> operator requires normalized vectors for accurate cosine distance
		embedding = postgres.NormalizeVector(embedding)

		// Store to database
		knowledgeChunk := &storage_models.KnowledgeChunk{
			TenantID:         tenantID,
			Content:          chunk.Text,
			Embedding:        embedding,
			EmbeddingModel:   kb.config.EmbeddingModel,
			EmbeddingVersion: 1,
			EmbeddingStatus:  "completed",
			SourceType:       "document",
			Source:           docPath,
			DocumentID:       docID,
			ChunkIndex:       chunk.Index,
			ContentHash:      chunk.Hash,
			AccessCount:      0,
		}

		if err := kb.repo.Create(ctx, knowledgeChunk); err != nil {
			log.Printf("Failed to save chunk %d: %v", i, err)
			continue
		}

		successCount++
	}

	log.Printf("Successfully imported %d/%d chunks", successCount, len(chunks))

	if successCount == 0 {
		return "", fmt.Errorf("failed to import any chunks")
	}

	return docID, nil
}

// Search retrieve knowledge base using simple vector similarity
func (kb *KnowledgeBase) Search(ctx context.Context, tenantID, question string) ([]*SearchResult, error) {
	log.Printf("Searching for: %s (tenant: %s)", question, tenantID)

	// Use SimpleRetrievalService for pure vector similarity search
	// This follows ChromaDB's simple approach: direct vector similarity without complex weights
	results, err := kb.retrieval.Search(ctx, tenantID, question)
	if err != nil {
		log.Printf("Search error: %v", err)
		return nil, err
	}

	log.Printf("Search returned %d results", len(results))
	for i, result := range results {
		log.Printf("  Result %d: score=%.3f, source=%s, content=%s", i, result.Score, result.Source, truncateString(result.Content, 50))
	}

	// Convert to local SearchResult type for backward compatibility
	var localResults []*SearchResult
	for _, r := range results {
		localResults = append(localResults, &SearchResult{
			Content: r.Content,
			Source:  r.Source,
			Score:   r.Score,
		})
	}

	return localResults, nil
}

// truncateString truncate string for log output
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ListDocuments list all documents
func (kb *KnowledgeBase) ListDocuments(ctx context.Context, tenantID string) ([]*DocumentInfo, error) {
	query := `
		SELECT document_id, source, COUNT(*) as chunk_count
		FROM knowledge_chunks_1024
		WHERE tenant_id = $1
		GROUP BY document_id, source
		ORDER BY MAX(created_at) DESC
	`

	rows, err := kb.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Fatal("Failed to close rows", err)
		}
	}()

	var docs []*DocumentInfo
	for rows.Next() {
		var doc DocumentInfo
		if err := rows.Scan(&doc.ID, &doc.Source, &doc.ChunkCount); err != nil {
			continue
		}
		docs = append(docs, &doc)
	}

	return docs, nil
}

// DeleteDocument delete document
func (kb *KnowledgeBase) DeleteDocument(ctx context.Context, tenantID, documentID string) error {
	query := `DELETE FROM knowledge_chunks_1024 WHERE tenant_id = $1 AND document_id = $2`

	result, err := kb.pool.Exec(ctx, query, tenantID, documentID)
	if err != nil {
		return fmt.Errorf("delete document: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("document not found")
	}

	return nil
}

// StartChat start interactive Q&A
func (kb *KnowledgeBase) StartChat(ctx context.Context, tenantID string) {
	scanner := NewTextScanner()

	for {
		fmt.Print("\nYou: ")
		question, err := scanner.ReadLine()
		if err != nil {
			if err == io.EOF {
				log.Println("\nGoodbye!")
			} else {
				log.Printf("Error reading input: %v", err)
			}
			break
		}

		question = strings.TrimSpace(question)
		if question == "" {
			continue
		}

		if question == "exit" || question == "quit" {
			log.Println("Goodbye!")
			break
		}

		// Set timeout for each retrieval operation (30 seconds)
		searchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		results, err := kb.Search(searchCtx, tenantID, question)
		cancel()

		if err != nil {
			if err == context.DeadlineExceeded {
				log.Println("Search timeout. Please try again.")
			} else {
				log.Printf("Search failed: %v", err)
			}
			continue
		}

		// Display results
		fmt.Printf("\nFound %d relevant results:\n", len(results))
		for i, result := range results {
			fmt.Printf("\n[%d] Score: %.3f\n", i+1, result.Score)
			fmt.Printf("Content: %s\n", result.Content)
			fmt.Printf("Source: %s\n", result.Source)
		}

		// If no results
		if len(results) == 0 {
			fmt.Println("No relevant information found in the knowledge base.")
		}
	}
}

// Close close connection
func (kb *KnowledgeBase) Close() error {
	return kb.pool.Close()
}

// chunkDocument chunk document
func (kb *KnowledgeBase) chunkDocument(content string, chunkSize, chunkOverlap int) []*Chunk {
	var chunks []*Chunk
	runes := []rune(content)
	contentLength := len(runes)

	for i := 0; i < contentLength; i += (chunkSize - chunkOverlap) {
		end := i + chunkSize
		if end > contentLength {
			end = contentLength
		}

		chunkText := string(runes[i:end])
		if strings.TrimSpace(chunkText) == "" {
			continue
		}

		chunks = append(chunks, &Chunk{
			Index: len(chunks),
			Text:  chunkText,
			Hash:  kb.generateHash(chunkText),
		})

		if end >= contentLength {
			break
		}
	}

	return chunks
}

// generateHash generate content hash (using SHA256)
func (kb *KnowledgeBase) generateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// loadConfig load configuration file
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Set default values
	setDefaults(&config)

	return &config, nil
}

// setDefaults set configuration default values
func setDefaults(config *Config) {
	if config.Database.Host == "" {
		config.Database.Host = "localhost"
	}
	if config.Database.Port == 0 {
		config.Database.Port = 5433
	}
	if config.Database.User == "" {
		config.Database.User = "postgres"
	}
	if config.Database.Database == "" {
		config.Database.Database = "goagent"
	}
	if config.EmbeddingServiceURL == "" {
		config.EmbeddingServiceURL = "http://localhost:8000"
	}
	if config.EmbeddingModel == "" {
		config.EmbeddingModel = "e5-large-v2"
	}
	if config.Knowledge.ChunkSize == 0 {
		config.Knowledge.ChunkSize = 1000
	}
	if config.Knowledge.ChunkOverlap == 0 {
		config.Knowledge.ChunkOverlap = 100
	}
	if config.Knowledge.TopK == 0 {
		config.Knowledge.TopK = 5
	}
	if config.Knowledge.MinScore == 0 {
		config.Knowledge.MinScore = 0.6
	}
}

// DocumentInfo document information
type DocumentInfo struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	ChunkCount int    `json:"chunk_count"`
}

// TextScanner simple text scanner
type TextScanner struct {
	scanner *bufio.Scanner
}

func NewTextScanner() *TextScanner {
	return &TextScanner{
		scanner: bufio.NewScanner(os.Stdin),
	}
}

func (s *TextScanner) ReadLine() (string, error) {
	if !s.scanner.Scan() {
		if err := s.scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return s.scanner.Text(), nil
}

// printUsage print usage instructions
func printUsage() {
	fmt.Println("Knowledge Base - Local Document Q&A System")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run main.go --save <doc_path>     Import a document")
	fmt.Println("  go run main.go --chat                Start interactive chat")
	fmt.Println("  go run main.go --list                List all documents")
	fmt.Println("  go run main.go --delete <doc_id>     Delete a document")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --tenant <id>    Tenant ID (default: default)")
	fmt.Println("  --config <path>  Config file path (default: config.yaml)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run main.go --save README.md")
	fmt.Println("  go run main.go --chat --tenant user123")
	fmt.Println("  go run main.go --list")
}
