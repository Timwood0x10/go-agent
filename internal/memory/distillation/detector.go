// Package distillation provides memory distillation functionality for agent experience extraction.
package distillation

import (
	"strings"
)

// IsProblem detects if a user message represents a genuine problem or question.
// It checks for problem-related keywords and question marks to avoid extracting
// non-problematic conversations like "thanks", "ok", "got it".
//
// Args:
//
//	text - the user message to analyze.
//
// Returns:
//
//	true if the text appears to be a problem or question, false otherwise.
func IsProblem(text string) bool {
	if text == "" {
		return false
	}

	lower := strings.TrimSpace(strings.ToLower(text))

	// Negative keywords - these should NOT be treated as problems (English and Chinese)
	negativeKeywords := []string{
		// English acknowledgments
		"thanks", "thank you", "ok", "okay", "got it", "understood",
		"alright", "sure", "fine", "good", "great", "perfect",
		"awesome", "excellent", "yes", "no", "maybe", "right",
		"correct", "agree", "cool", "nice", "sounds good",
		"that works", "makes sense", "got it, thanks", "thanks for the",
		"you're welcome", "glad i could", "appreciate", "welcome",
		"show me", "tell me", "what's happening", "what is this",
		// Chinese acknowledgments
		"谢谢", "感谢", "好的", "没问题", "明白了", "知道了",
		"好的的", "行", "可以", "不错", "很棒", "太好了",
		"完美", "优秀", "是的", "对", "正确", "同意",
		"酷", "很好", "听起来不错", "有道理", "收到了",
		"欢迎", "不用谢", "请", "请问", "你好", "hi",
		"hello", "再见", "拜拜",
	}

	// Check negative keywords - return false immediately if matched
	for _, keyword := range negativeKeywords {
		if lower == keyword || strings.HasPrefix(lower, keyword+" ") || strings.HasSuffix(lower, " "+keyword) {
			return false
		}
	}

	// Problem-related keywords (must be more specific than casual terms) - English and Chinese
	problemKeywords := []string{
		// English problem keywords
		"error", "issue", "problem", "fix", "help", "unable",
		"cannot", "can't", "fail", "failed", "broken", "wrong",
		"not working", "doesn't work", "won't work", "won't start",
		"won't", "bug", "crash", "exception", "panic",
		"stack trace", "leak", "timeout", "refused", "denied",
		"missing", "undefined", "null", "invalid",
		// Chinese problem keywords
		"错误", "问题", "故障", "怎么", "如何", "怎么办",
		"无法", "不能", "失败", "崩溃", "异常", "恐慌",
		"超时", "拒绝", "缺失", "未定义", "无效", "出错",
		"修复", "解决", "调试", "排查", "帮忙", "求助",
		"为什么", "为何", "怎样", "怎么弄", "怎么做",
		"什么", "哪里", "哪个", "如何", "是否", "有没有",
		"为什么", "为啥", "为啥子",
	}

	for _, keyword := range problemKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}

	// Check for question mark (but filter out rhetorical questions) - supports both English and Chinese
	if strings.Contains(text, "?") || strings.Contains(text, "？") {
		// Filter out questions that are just acknowledgments
		questionExclusions := []string{
			// English exclusions
			"can you?", "could you?", "would you?", "right?",
			"ok?", "sure?", "yes?", "no?",
			// Chinese exclusions
			"好吗？", "可以吗？", "是吗？", "对吗？", "没错？",
			"明白吗？", "知道吗？", "好的？", "行吗？",
		}
		for _, exclusion := range questionExclusions {
			if strings.HasSuffix(lower, exclusion) {
				return false
			}
		}
		return true
	}

	return false
}

// QuestionDetector detects questions in conversations.
type QuestionDetector struct {
	// Configurable sensitivity (0.0 to 1.0)
	sensitivity float64
}

// NewQuestionDetector creates a new QuestionDetector with default sensitivity.
func NewQuestionDetector() *QuestionDetector {
	return &QuestionDetector{
		sensitivity: 0.7,
	}
}

// Detect checks if a message is a question.
//
// Args:
//
//	text - the message to analyze.
//
// Returns:
//
//	true if the message is likely a question.
func (d *QuestionDetector) Detect(text string) bool {
	return IsProblem(text)
}
