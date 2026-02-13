package template

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	textTemplate "text/template"
	"time"

	"github.com/Ruseigha/SendFlix/pkg/logger"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Engine handles template rendering with advanced features
type Engine struct {
	htmlTemplates map[string]*template.Template
	textTemplates map[string]*textTemplate.Template
	funcMap       template.FuncMap
	translators   map[string]*message.Printer
	templatesDir  string
	cacheEnabled  bool
	mu            sync.RWMutex
	logger        logger.Logger
}

// Config contains template engine configuration
type Config struct {
	TemplatesDir string
	CacheEnabled bool
	Languages    []string
}

// NewEngine creates new template engine
func NewEngine(config Config, logger logger.Logger) *Engine {
	engine := &Engine{
		htmlTemplates: make(map[string]*template.Template),
		textTemplates: make(map[string]*textTemplate.Template),
		translators:   make(map[string]*message.Printer),
		templatesDir:  config.TemplatesDir,
		cacheEnabled:  config.CacheEnabled,
		logger:        logger,
	}

	// Initialize custom functions
	engine.funcMap = engine.createFuncMap()

	// Initialize translators
	engine.initTranslators(config.Languages)

	return engine
}

// RenderHTML renders HTML template
func (e *Engine) RenderHTML(templateName string, data map[string]interface{}) (string, error) {
	e.logger.Debug("rendering HTML template", "name", templateName)

	tmpl, err := e.getHTMLTemplate(templateName)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// RenderText renders text template
func (e *Engine) RenderText(templateName string, data map[string]interface{}) (string, error) {
	e.logger.Debug("rendering text template", "name", templateName)

	tmpl, err := e.getTextTemplate(templateName)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// getHTMLTemplate gets or loads HTML template
func (e *Engine) getHTMLTemplate(name string) (*template.Template, error) {
	// Check cache
	if e.cacheEnabled {
		e.mu.RLock()
		if tmpl, exists := e.htmlTemplates[name]; exists {
			e.mu.RUnlock()
			return tmpl, nil
		}
		e.mu.RUnlock()
	}

	// Load and parse template
	tmpl, err := e.parseHTMLTemplate(name)
	if err != nil {
		return nil, err
	}

	// Cache if enabled
	if e.cacheEnabled {
		e.mu.Lock()
		e.htmlTemplates[name] = tmpl
		e.mu.Unlock()
	}

	return tmpl, nil
}

// getTextTemplate gets or loads text template
func (e *Engine) getTextTemplate(name string) (*textTemplate.Template, error) {
	// Check cache
	if e.cacheEnabled {
		e.mu.RLock()
		if tmpl, exists := e.textTemplates[name]; exists {
			e.mu.RUnlock()
			return tmpl, nil
		}
		e.mu.RUnlock()
	}

	// Load and parse template
	tmpl, err := e.parseTextTemplate(name)
	if err != nil {
		return nil, err
	}

	// Cache if enabled
	if e.cacheEnabled {
		e.mu.Lock()
		e.textTemplates[name] = tmpl
		e.mu.Unlock()
	}

	return tmpl, nil
}

// parseHTMLTemplate loads and parses HTML template with layout
func (e *Engine) parseHTMLTemplate(name string) (*template.Template, error) {
	// Check if layout exists
	layoutPath := filepath.Join(e.templatesDir, "layouts", "base.html")
	templatePath := filepath.Join(e.templatesDir, name+".html")

	tmpl := template.New(name).Funcs(e.funcMap)

	// Parse layout if exists
	if fileExists(layoutPath) {
		tmpl, err := tmpl.ParseFiles(layoutPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layout: %w", err)
		}
		return tmpl.ParseFiles(templatePath)
	}

	// Parse template only
	return tmpl.ParseFiles(templatePath)
}

// parseTextTemplate loads and parses text template
func (e *Engine) parseTextTemplate(name string) (*textTemplate.Template, error) {
	templatePath := filepath.Join(e.templatesDir, name+".txt")

	tmpl := textTemplate.New(name).Funcs(textTemplate.FuncMap(e.funcMap))
	return tmpl.ParseFiles(templatePath)
}

// createFuncMap creates custom template functions
func (e *Engine) createFuncMap() template.FuncMap {
	return template.FuncMap{
		// String functions
		"upper":    strings.ToUpper,
		"lower":    strings.ToLower,
		"title":    strings.Title,
		"truncate": e.truncate,
		"contains": strings.Contains,
		"replace":  strings.ReplaceAll,

		// Number functions
		"formatCurrency": e.formatCurrency,
		"formatNumber":   e.formatNumber,
		"add":            e.add,
		"subtract":       e.subtract,
		"multiply":       e.multiply,
		"divide":         e.divide,

		// Date functions
		"formatDate":     e.formatDate,
		"formatDateTime": e.formatDateTime,
		"now":            time.Now,
		"daysAgo":        e.daysAgo,
		"daysUntil":      e.daysUntil,

		// Logic functions
		"default": e.defaultValue,
		"ternary": e.ternary,
		"eq":      e.eq,
		"ne":      e.ne,
		"lt":      e.lt,
		"gt":      e.gt,

		// i18n functions
		"t": e.translate,

		// Misc functions
		"safe": e.safeHTML,
		"url":  e.urlEncode,
		"json": e.toJSON,
	}
}

// String functions

func (e *Engine) truncate(length int, s string) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// Number functions

func (e *Engine) formatCurrency(amount float64, currency string) string {
	switch currency {
	case "USD":
		return fmt.Sprintf("$%.2f", amount)
	case "EUR":
		return fmt.Sprintf("€%.2f", amount)
	case "GBP":
		return fmt.Sprintf("£%.2f", amount)
	default:
		return fmt.Sprintf("%.2f %s", amount, currency)
	}
}

func (e *Engine) formatNumber(n interface{}) string {
	switch v := n.(type) {
	case int:
		return formatWithCommas(v)
	case int64:
		return formatWithCommas(int(v))
	case float64:
		return formatWithCommas(int(v))
	default:
		return fmt.Sprintf("%v", n)
	}
}

func formatWithCommas(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, c)
	}
	return string(result)
}

func (e *Engine) add(a, b interface{}) float64 {
	return toFloat64(a) + toFloat64(b)
}

func (e *Engine) subtract(a, b interface{}) float64 {
	return toFloat64(a) - toFloat64(b)
}

func (e *Engine) multiply(a, b interface{}) float64 {
	return toFloat64(a) * toFloat64(b)
}

func (e *Engine) divide(a, b interface{}) float64 {
	if toFloat64(b) == 0 {
		return 0
	}
	return toFloat64(a) / toFloat64(b)
}

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	case float32:
		return float64(n)
	default:
		return 0
	}
}

// Date functions

func (e *Engine) formatDate(date time.Time) string {
	return date.Format("January 2, 2006")
}

func (e *Engine) formatDateTime(date time.Time) string {
	return date.Format("January 2, 2006 3:04 PM")
}

func (e *Engine) daysAgo(date time.Time) int {
	return int(time.Since(date).Hours() / 24)
}

func (e *Engine) daysUntil(date time.Time) int {
	return int(time.Until(date).Hours() / 24)
}

// Logic functions

func (e *Engine) defaultValue(defaultVal, val interface{}) interface{} {
	if val == nil || val == "" {
		return defaultVal
	}
	return val
}

func (e *Engine) ternary(condition bool, trueVal, falseVal interface{}) interface{} {
	if condition {
		return trueVal
	}
	return falseVal
}

func (e *Engine) eq(a, b interface{}) bool {
	return a == b
}

func (e *Engine) ne(a, b interface{}) bool {
	return a != b
}

func (e *Engine) lt(a, b interface{}) bool {
	return toFloat64(a) < toFloat64(b)
}

func (e *Engine) gt(a, b interface{}) bool {
	return toFloat64(a) > toFloat64(b)
}

// i18n functions

func (e *Engine) initTranslators(languages []string) {
	for _, lang := range languages {
		tag := language.MustParse(lang)
		e.translators[lang] = message.NewPrinter(tag)
	}
}

func (e *Engine) translate(key string, lang string) string {
	if printer, ok := e.translators[lang]; ok {
		return printer.Sprintf(key)
	}
	return key
}

// LoadTranslations loads translation files
func (e *Engine) LoadTranslations(translationsDir string) error {
	// Load translation YAML files
	// Implementation depends on i18n library
	return nil
}

// Misc functions

func (e *Engine) safeHTML(s string) template.HTML {
	return template.HTML(s)
}

func (e *Engine) urlEncode(s string) string {
	return strings.ReplaceAll(s, " ", "%20")
}

func (e *Engine) toJSON(v interface{}) string {
	// Simple JSON conversion
	return fmt.Sprintf("%v", v)
}

// ClearCache clears template cache
func (e *Engine) ClearCache() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.htmlTemplates = make(map[string]*template.Template)
	e.textTemplates = make(map[string]*textTemplate.Template)

	e.logger.Info("template cache cleared")
}

// Helper functions

func fileExists(path string) bool {
	_, err := ioutil.ReadFile(path)
	return err == nil
}