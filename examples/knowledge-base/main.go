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
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"goagent/internal/llm"
	"goagent/internal/memory"
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

	LLM struct {
		Provider  string `yaml:"provider"`
		APIKey    string `yaml:"api_key"`
		BaseURL   string `yaml:"base_url"`
		Model     string `yaml:"model"`
		Timeout   int    `yaml:"timeout"`
		MaxTokens int    `yaml:"max_tokens"`
	} `yaml:"llm"`

	Memory struct {
		Enabled               bool `yaml:"enabled"`
		MaxHistory            int  `yaml:"max_history"`
		MaxSessions           int  `yaml:"max_sessions"`
		EnableDistillation    bool `yaml:"enable_distillation"`
		DistillationThreshold int  `yaml:"distillation_threshold"`
	} `yaml:"memory"`

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
	config       *Config
	pool         *postgres.Pool
	repo         *repositories.KnowledgeRepository
	embedding    *embedding.EmbeddingClient
	llmClient    *llm.Client
	retrieval    *services.SimpleRetrievalService
	memory       memory.MemoryManager
	sessionID    string
	messageCount int
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
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Create knowledge base
	kb, err := NewKnowledgeBase(config)
	if err != nil {
		slog.Error("Failed to create knowledge base", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := kb.Close(); err != nil {
			slog.Error("Failed to close knowledge base", "error", err)
			os.Exit(1)
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
				slog.Error("Import timeout (5 minutes exceeded)")
				os.Exit(1)
			}
			slog.Error("Failed to import document", "error", err)
			os.Exit(1)
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
				slog.Error("List timeout")
				os.Exit(1)
			}
			slog.Error("Failed to list documents", "error", err)
			os.Exit(1)
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
				slog.Error("Delete timeout")
				os.Exit(1)
			}
			slog.Error("Failed to delete document", "error", err)
			os.Exit(1)
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

	// Create LLM client for RAG (optional, may be nil)
	var llmClient *llm.Client
	if config.LLM.Provider != "" && config.LLM.Model != "" {
		llmConfig := &llm.Config{
			Provider: config.LLM.Provider,
			APIKey:   config.LLM.APIKey,
			BaseURL:  config.LLM.BaseURL,
			Model:    config.LLM.Model,
			Timeout:  config.LLM.Timeout,
		}
		var err error
		llmClient, err = llm.NewClient(llmConfig)
		if err != nil {
			log.Printf("Failed to create LLM client: %v, RAG will be disabled", err)
			llmClient = nil
		}
	}

	// Create memory manager for conversation history
	var memManager memory.MemoryManager
	if config.Memory.Enabled {
		memConfig := &memory.MemoryConfig{
			Enabled:        true,
			Storage:        "memory",
			MaxHistory:     config.Memory.MaxHistory,
			MaxSessions:    config.Memory.MaxSessions,
			SessionTTL:     24 * time.Hour,
			TaskTTL:        7 * 24 * time.Hour,
			VectorDim:      128,
			EnablePostgres: false,
		}
		var err error
		memManager, err = memory.NewMemoryManager(memConfig)
		if err != nil {
			log.Printf("Failed to create memory manager: %v", err)
			memManager = nil
		}
	}

	return &KnowledgeBase{
		config:       config,
		pool:         pool,
		repo:         kbRepo,
		embedding:    embeddingClient,
		llmClient:    llmClient,
		retrieval:    retrievalService,
		memory:       memManager,
		sessionID:    "",
		messageCount: 0,
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
		log.Printf("Processing chunk %d/%d...", i+1, len(chunks))

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

// GenerateAnswer generates a response based on retrieved documents using LLM.
// This implements the complete RAG pipeline: retrieve → generate → validate.
// Args:
// ctx - operation context.
// tenantID - tenant identifier for isolation.
// question - user question.
// Returns generated answer or error if generation fails.
func (kb *KnowledgeBase) GenerateAnswer(ctx context.Context, tenantID, question string) (string, error) {
	// Step 0: Add user question to memory
	if kb.memory != nil && kb.sessionID != "" {
		_ = kb.memory.AddMessage(ctx, kb.sessionID, "user", question)
	}

	// Step 1: Determine if RAG is needed using LLM
	needsRAG := true
	if kb.llmClient != nil {
		ragPrompt := fmt.Sprintf(`You are a knowledge retrieval assistant. Determine if the following question requires searching the knowledge base.

Question: %s

Answer with "YES" if this question needs knowledge base search:
- Asking about specific documentation, code rules, technical specifications
- Questions about facts, procedures, configurations
- Technical queries about code, systems, or frameworks

Answer with "NO" if this is general conversation:
- Greetings (hello, hi, 你好)
- Personal introductions (my name is..., 我叫...)
- General chat (how are you, 谢谢)
- Questions about personal information (what's your name, 还记得我的名字吗)

Answer (YES/NO only):`, question)

		ragCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		ragResponse, err := kb.llmClient.Generate(ragCtx, ragPrompt)
		cancel()

		if err == nil {
			needsRAG = strings.Contains(strings.ToUpper(ragResponse), "YES")
			log.Printf("RAG check: %s -> needs RAG: %v", question, needsRAG)
		}
	}

	var answer string
	if needsRAG {
		// Step 2: Retrieve relevant documents
		results, err := kb.Search(ctx, tenantID, question)
		if err != nil {
			return "", fmt.Errorf("retrieve documents: %w", err)
		}

		// If no documents found, inform user
		if len(results) == 0 {
			answer = "No relevant information found in the knowledge base. Please try rephrasing your question."
		} else {
			// Step 3: Build context from retrieved documents
			var contextBuilder strings.Builder
			for i, result := range results {
				fmt.Fprintf(&contextBuilder, "[Document %d - Score: %.3f]\n%s\n\n", i+1, result.Score, result.Content)
			}

			// Step 4: Build conversation history context
			var historyContext strings.Builder
			if kb.memory != nil && kb.sessionID != "" {
				messages, err := kb.memory.GetMessages(ctx, kb.sessionID)
				if err == nil && len(messages) > 0 {
					// Get last 5 messages for context
					start := len(messages) - 5
					if start < 0 {
						start = 0
					}
					for _, msg := range messages[start:] {
						fmt.Fprintf(&historyContext, "%s: %s\n", msg.Role, msg.Content)
					}
				}
			}

			// Step 5: Generate answer with LLM
			if kb.llmClient != nil {
				prompt := fmt.Sprintf(`You are a helpful assistant that answers questions based on the provided knowledge base and conversation history.

User Question: %s

Conversation History:
%s

Knowledge Base Context:
%s

CRITICAL INSTRUCTIONS:
1. Answer the user's question based ONLY on the provided knowledge base context.
2. If the user's question contains INCORRECT ASSUMPTIONS or MISUNDERSTANDINGS, POLITELY CORRECT them with FACTS from the context.
3. DO NOT simply agree with the user if they are wrong - use facts to correct them.
4. DO NOT make up information that is not in the context.
5. If the context doesn't contain the answer, say "I don't have enough information to answer this question."
6. Be concise and direct.
7. Cite the relevant document numbers (e.g., [Document 1]) when using information from specific documents.

Answer:`, question, historyContext.String(), contextBuilder.String())

				// Call LLM with timeout
				genCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
				var genErr error
				answer, genErr = kb.llmClient.Generate(genCtx, prompt)
				cancel()

				if genErr != nil {
					log.Printf("LLM generation failed: %v, falling back to raw results", genErr)
					// Fall back to showing raw results if LLM fails
					answer = kb.formatRawResults(results)
				}
			} else {
				// Step 6: If LLM client is not available, format raw results
				answer = kb.formatRawResults(results)
			}
		}
	} else {
		// General conversation without RAG
		if kb.llmClient != nil {
			prompt := fmt.Sprintf(`You are a helpful assistant. Respond to the user's question naturally.

User: %s

Assistant:`, question)

			genCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			var genErr error
			answer, genErr = kb.llmClient.Generate(genCtx, prompt)
			cancel()

			if genErr != nil {
				answer = "I'm sorry, I'm having trouble generating a response right now. Please try again."
			}
		} else {
			answer = "LLM not configured. Please configure the llm section in config.yaml."
		}
	}

	// Step 7: Add assistant answer to memory
	if kb.memory != nil && kb.sessionID != "" {
		_ = kb.memory.AddMessage(ctx, kb.sessionID, "assistant", answer)

		// Step 8: Check for distillation threshold
		kb.messageCount++
		if kb.config.Memory.EnableDistillation && kb.messageCount >= kb.config.Memory.DistillationThreshold {
			log.Printf("🎯 [记忆蒸馏] 对话轮数达到阈值 (%d/%d)，触发记忆蒸馏...",
				kb.messageCount, kb.config.Memory.DistillationThreshold)
			kb.distillMemory(ctx, tenantID)
			kb.messageCount = 0
		}
	}

	return answer, nil
}

// formatRawResults formats search results for display when LLM is not available.
func (kb *KnowledgeBase) formatRawResults(results []*SearchResult) string {
	var output strings.Builder
	fmt.Fprintf(&output, "Found %d relevant documents:\n\n", len(results))
	for i, result := range results {
		fmt.Fprintf(&output, "[Document %d - Score: %.3f]\n%s\n\n", i+1, result.Score, result.Content)
	}
	output.WriteString("Please configure LLM settings in config.yaml to enable natural language answers.")
	return output.String()
}

// distillMemory performs memory distillation when threshold is reached.
// It extracts key information from conversation history and stores it in the knowledge base.
func (kb *KnowledgeBase) distillMemory(ctx context.Context, tenantID string) {
	if kb.memory == nil || kb.sessionID == "" {
		log.Printf("⚠️  Memory not available for distillation")
		return
	}

	log.Printf("🔄 [记忆蒸馏] 开始蒸馏会话: %s", kb.sessionID)

	// Get conversation history
	messages, err := kb.memory.GetMessages(ctx, kb.sessionID)
	if err != nil || len(messages) == 0 {
		log.Printf("⚠️  [记忆蒸馏] 没有消息需要蒸馏: %v", err)
		return
	}

	log.Printf("📊 [记忆蒸馏] 找到 %d 条消息需要蒸馏", len(messages))

	// Build conversation summary
	var summary strings.Builder
	summary.WriteString("Conversation Summary:\n\n")
	for _, msg := range messages {
		fmt.Fprintf(&summary, "%s: %s\n", msg.Role, msg.Content)
	}

	summaryText := summary.String()

	log.Printf("📝 [记忆蒸馏] 蒸馏内容预览 (%d 字符): %s",
		len(summaryText), truncateString(summaryText, 100))

	// Generate embedding for the distilled memory
	embedding, err := kb.embedding.EmbedWithPrefix(ctx, summaryText, "memory:")
	if err != nil {
		log.Printf("❌ [记忆蒸馏] 生成嵌入失败: %v", err)
		return
	}

	// Normalize embedding
	embedding = postgres.NormalizeVector(embedding)
	log.Printf("🔢 [记忆蒸馏] 嵌入向量维度: %d", len(embedding))

	// Generate document ID
	docID := uuid.New().String()

	// Store distilled memory in knowledge base
	distilledChunk := &storage_models.KnowledgeChunk{
		TenantID:         tenantID,
		Content:          summaryText,
		Embedding:        embedding,
		EmbeddingModel:   kb.config.EmbeddingModel,
		EmbeddingVersion: 1,
		EmbeddingStatus:  "completed",
		SourceType:       "distilled",
		Source:           fmt.Sprintf("memory:%s", kb.sessionID),
		DocumentID:       docID,
		ChunkIndex:       0,
		ContentHash:      kb.generateHash(summaryText),
		AccessCount:      0,
	}

	if err := kb.repo.Create(ctx, distilledChunk); err != nil {
		log.Printf("❌ [记忆蒸馏] 存储蒸馏记忆失败: %v", err)
		return
	}

	log.Printf("✅ [记忆蒸馏] 蒸馏完成！")
	log.Printf("   📄 文档ID: %s", docID)
	log.Printf("   📏 内容长度: %d 字符", len(summaryText))
	log.Printf("   🧠 嵌入维度: %d", len(embedding))
	log.Printf("   💾 存储位置: knowledge_chunks_1024")
	log.Printf("   🔍 可通过向量检索回放记忆")
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
			slog.Error("Failed to close rows", "error", err)
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

// StartChat start interactive Q&A with RAG and memory
func (kb *KnowledgeBase) StartChat(ctx context.Context, tenantID string) {
	scanner := NewTextScanner()

	// Check if LLM is configured
	llmEnabled := kb.llmClient != nil && kb.llmClient.IsEnabled()
	if llmEnabled {
		log.Println("LLM enabled - Using RAG (Retrieval + Generation) mode")
	} else {
		log.Println("LLM not configured - Using retrieval-only mode")
		log.Println("To enable LLM answers, configure llm section in config.yaml")
	}

	// Check if memory is enabled
	memoryEnabled := kb.memory != nil
	if memoryEnabled {
		log.Println("Memory enabled - Conversation history and distillation supported")
		// Start memory manager
		if err := kb.memory.Start(ctx); err != nil {
			log.Printf("Failed to start memory manager: %v", err)
		}
		// Create session
		sessionID, err := kb.memory.CreateSession(ctx, tenantID)
		if err != nil {
			log.Printf("Failed to create session: %v", err)
		} else {
			kb.sessionID = sessionID
			log.Printf("Session created: %s", sessionID)
		}
	}

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

		// Set timeout for each RAG operation (120 seconds)
		ragCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
		answer, err := kb.GenerateAnswer(ragCtx, tenantID, question)
		cancel()

		if err != nil {
			if err == context.DeadlineExceeded {
				log.Println("Response timeout. Please try again.")
			} else {
				log.Printf("Failed to generate answer: %v", err)
			}
			continue
		}

		// Display generated answer
		fmt.Printf("\nAssistant:\n%s\n", answer)
	}

	// Cleanup: stop memory manager
	if memoryEnabled && kb.memory != nil {
		if err := kb.memory.Stop(ctx); err != nil {
			log.Printf("Failed to stop memory manager: %v", err)
		}
	}
}

// Close close connection
func (kb *KnowledgeBase) Close() error {
	// Stop memory manager if running
	if kb.memory != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = kb.memory.Stop(ctx)
	}
	// Close database connection
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
	if config.LLM.Provider == "" {
		config.LLM.Provider = "openrouter"
	}
	if config.LLM.BaseURL == "" {
		config.LLM.BaseURL = "https://openrouter.ai/api/v1"
	}
	if config.LLM.Model == "" {
		config.LLM.Model = "meta-llama/llama-3.1-8b-instruct"
	}
	if config.LLM.Timeout == 0 {
		config.LLM.Timeout = 60
	}
	if config.LLM.MaxTokens == 0 {
		config.LLM.MaxTokens = 2048
	}
	if config.Memory.MaxHistory == 0 {
		config.Memory.MaxHistory = 10
	}
	if config.Memory.MaxSessions == 0 {
		config.Memory.MaxSessions = 100
	}
	if config.Memory.DistillationThreshold == 0 {
		config.Memory.DistillationThreshold = 5
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
