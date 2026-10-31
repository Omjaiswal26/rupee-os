package handlers

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rupee-os/rupee-os/internal/classification"
	"github.com/rupee-os/rupee-os/internal/config"
	"github.com/rupee-os/rupee-os/internal/services"
	"github.com/rupee-os/rupee-os/internal/validation"
	"gorm.io/gorm"
)

type API struct {
	Imports     *services.ImportService
	Txns        *services.TransactionService
	MonthHealth *services.HealthService
	Cfg         config.Config
	Log         *slog.Logger
}

func (a *API) Register(r *gin.Engine) {
	r.GET("/health", a.Health)

	r.POST("/imports", a.CreateImport)
	r.GET("/imports", a.ListImports)
	r.GET("/imports/:id", a.GetImport)

	r.GET("/dashboard", a.YearDashboard)
	r.GET("/dashboard/health", a.GetYearHealth)
	r.POST("/dashboard/health", a.GenerateYearHealth)

	r.GET("/months", a.ListMonths)
	r.GET("/months/calendar", a.MonthCalendar)
	r.GET("/months/:month/summary", a.MonthSummary)
	r.GET("/months/:month/transactions", a.MonthTransactions)
	r.GET("/months/:month/health", a.GetMonthHealth)
	r.POST("/months/:month/health", a.GenerateMonthHealth)
	r.POST("/months/:month/reclassify", a.ReclassifyMonth)
	r.DELETE("/months/:month", a.ClearMonth)

	r.GET("/transactions", a.ListTransactions)
	r.GET("/transactions/review", a.ListReview)
	r.PATCH("/transactions/:id/classification", a.CorrectClassification)

	r.GET("/rules", a.ListRules)
	r.GET("/meta/categories", a.ListCategories)
	r.GET("/meta/ollama", a.OllamaStatus)
}

func (a *API) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "api"})
}

func (a *API) OllamaStatus(c *gin.Context) {
	st := classification.CheckOllamaModels(
		c.Request.Context(),
		a.Cfg.OllamaBaseURL,
		a.Cfg.OllamaModel,
		a.Cfg.OllamaHealthModel,
	)
	c.JSON(http.StatusOK, st)
}

func (a *API) CreateImport(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required (multipart field: file)", "code": "MISSING_FILE"})
		return
	}
	maxBytes := int64(a.Cfg.MaxUploadMB) * 1024 * 1024
	if file.Size > maxBytes {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "file too large",
			"code":   "FILE_TOO_LARGE",
			"max_mb": a.Cfg.MaxUploadMB,
		})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".csv" && ext != ".xlsx" && ext != ".xls" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "only .xlsx or .csv bank statements are supported",
			"code":  validation.ErrCodeUnsupportedType,
		})
		return
	}

	tmp, err := os.CreateTemp("", "rupee-upload-*"+ext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not stage upload"})
		return
	}
	tmpPath := tmp.Name()
	defer func() {
		tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read upload"})
		return
	}
	defer src.Close()
	if _, err := io.Copy(tmp, src); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save upload"})
		return
	}
	_ = tmp.Close()

	// Multipart fields: file + optional password (never logged).
	_ = c.Request.ParseMultipartForm(maxBytes)
	password := strings.TrimSpace(c.PostForm("password"))
	if password == "" {
		password = strings.TrimSpace(c.Request.FormValue("password"))
	}
	replaceMonth := strings.TrimSpace(c.PostForm("replace_month"))
	a.Log.Info("import upload received",
		"filename", filepath.Base(file.Filename),
		"password_provided", password != "",
		"replace_month", replaceMonth != "",
	)
	imp, err := a.Imports.Create(c.Request.Context(), services.CreateImportInput{
		Filename:     file.Filename,
		TempPath:     tmpPath,
		Password:     password,
		ReplaceMonth: replaceMonth,
	})
	password = "" // drop from scope ASAP
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{
		"import": imp,
		"message": "Upload accepted. Processing runs in the background. " +
			"Each file must contain transactions for exactly one calendar month.",
	})
}

func (a *API) ListImports(c *gin.Context) {
	items, err := a.Imports.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list imports"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"imports": items})
}

func (a *API) GetImport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid import id"})
		return
	}
	imp, err := a.Imports.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "import not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get import"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"import": imp})
}

func (a *API) YearDashboard(c *gin.Context) {
	year, _ := strconv.Atoi(c.DefaultQuery("year", ""))
	if year == 0 {
		year = time.Now().UTC().Year()
	}
	dash, err := a.MonthHealth.YearDashboard(c.Request.Context(), year)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dash)
}

func (a *API) GetYearHealth(c *gin.Context) {
	year, _ := strconv.Atoi(c.DefaultQuery("year", ""))
	if year == 0 {
		year = time.Now().UTC().Year()
	}
	rep, err := a.MonthHealth.GetYearHealth(c.Request.Context(), year)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "year health not generated yet", "year": year})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rep)
}

func (a *API) GenerateYearHealth(c *gin.Context) {
	year, _ := strconv.Atoi(c.DefaultQuery("year", ""))
	if year == 0 {
		year = time.Now().UTC().Year()
	}
	rep, err := a.MonthHealth.GenerateYearHealth(c.Request.Context(), year)
	if err != nil {
		if errors.Is(err, classification.ErrHealthModelMissing) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": err.Error(),
				"code":  "HEALTH_MODEL_MISSING",
				"hint":  "run ./scripts/pull-ollama-model.sh",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rep)
}

func (a *API) ListMonths(c *gin.Context) {
	months, err := a.Txns.ListMonths(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list months"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"months": months})
}

func (a *API) MonthCalendar(c *gin.Context) {
	year, _ := strconv.Atoi(c.DefaultQuery("year", ""))
	if year == 0 {
		year = time.Now().UTC().Year()
	}
	cal, err := a.Txns.Calendar(c.Request.Context(), year)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cal)
}

func (a *API) ReclassifyMonth(c *gin.Context) {
	n, err := a.Imports.ReclassifyMonth(c.Request.Context(), c.Param("month"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{
		"month":       c.Param("month"),
		"reset_count": n,
		"message":     "Reclassification queued. Pending transactions will be classified locally.",
	})
}

func (a *API) GetMonthHealth(c *gin.Context) {
	month := c.Param("month")
	rep, err := a.MonthHealth.Get(c.Request.Context(), month)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "health report not generated yet", "month": month})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rep)
}

func (a *API) GenerateMonthHealth(c *gin.Context) {
	month := c.Param("month")
	rep, err := a.MonthHealth.Generate(c.Request.Context(), month)
	if err != nil {
		if errors.Is(err, classification.ErrHealthModelMissing) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": err.Error(),
				"code":  "HEALTH_MODEL_MISSING",
				"hint":  "run ./scripts/pull-ollama-model.sh",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rep)
}

func (a *API) ClearMonth(c *gin.Context) {
	month := c.Param("month")
	_ = a.MonthHealth.DeleteMonth(c.Request.Context(), month)
	if err := a.Imports.ClearMonth(c.Request.Context(), month); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"month": c.Param("month"), "cleared": true})
}

func (a *API) MonthSummary(c *gin.Context) {
	sum, err := a.Txns.Summary(c.Request.Context(), c.Param("month"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sum)
}

func (a *API) MonthTransactions(c *gin.Context) {
	a.listTxns(c, c.Param("month"))
}

func (a *API) ListTransactions(c *gin.Context) {
	a.listTxns(c, c.Query("month"))
}

func (a *API) listTxns(c *gin.Context, month string) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	sort := c.DefaultQuery("sort", "date")
	order := c.DefaultQuery("order", "desc")
	rows, total, err := a.Txns.List(c.Request.Context(), services.TransactionFilter{
		Month:    month,
		Query:    c.Query("q"),
		Category: c.Query("category"),
		Status:   c.Query("status"),
		Sort:     sort,
		Order:    order,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"transactions": rows,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
		"sort":         sort,
		"order":        order,
	})
}

func (a *API) ListReview(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	month := c.Query("month")
	rows, err := a.Txns.ListAttention(c.Request.Context(), month, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list review queue"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transactions": rows, "total": len(rows)})
}

func (a *API) CorrectClassification(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction id"})
		return
	}
	var in services.CorrectClassificationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body", "detail": err.Error()})
		return
	}
	res, err := a.Txns.Correct(c.Request.Context(), id, in)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"transaction":   res.Transaction,
		"applied_count": res.AppliedCount,
		"pattern":       res.Pattern,
	})
}

func (a *API) ListRules(c *gin.Context) {
	rules, err := a.Txns.ListRules(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func (a *API) ListCategories(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"categories": classification.AllowedCategories,
		"types":      classification.AllowedTypes,
	})
}

func writeServiceError(c *gin.Context, err error) {
	if ve, ok := err.(*validation.Error); ok {
		status := http.StatusBadRequest
		if ve.Code == validation.ErrCodeDuplicateImport {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": ve.Message, "code": ve.Code})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}
