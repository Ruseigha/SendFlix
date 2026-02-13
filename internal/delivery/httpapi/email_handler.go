package httpapi

import (
	"net/http"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/gin-gonic/gin"

	emailuc "github.com/Ruseigha/SendFlix/internal/usecase/email"
)

// sendEmail handles POST /api/v1/emails
func (s *Server) sendEmail(c *gin.Context) {
	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Build use case request
	ucReq := emailuc.SendEmailRequest{
		To:           req.To,
		CC:           req.CC,
		BCC:          req.BCC,
		From:         req.From,
		ReplyTo:      req.ReplyTo,
		Subject:      req.Subject,
		BodyHTML:     req.BodyHTML,
		BodyText:     req.BodyText,
		TemplateID:   req.TemplateID,
		TemplateData: req.TemplateData,
		Priority:     domain.EmailPriority(req.Priority),
		ProviderName: req.ProviderName,
		TrackOpens:   req.TrackOpens,
		TrackClicks:  req.TrackClicks,
		Metadata:     req.Metadata,
	}

	// Execute use case
	resp, err := s.useCases.SendEmail.Execute(c.Request.Context(), ucReq)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	// Respond
	c.JSON(http.StatusOK, SendEmailResponse{
		EmailID:           resp.EmailID,
		Status:            string(resp.Status),
		ProviderName:      resp.ProviderName,
		ProviderMessageID: resp.ProviderMessageID,
		CreatedAt:         resp.CreatedAt,
		SentAt:            resp.SentAt,
		Message:           resp.Message,
	})
}

// sendBulkEmails handles POST /api/v1/emails/bulk
func (s *Server) sendBulkEmails(c *gin.Context) {
	var req SendBulkEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Build use case request
	emails := make([]emailuc.SendEmailRequest, len(req.Emails))
	for i, e := range req.Emails {
		emails[i] = emailuc.SendEmailRequest{
			To:           e.To,
			CC:           e.CC,
			BCC:          e.BCC,
			From:         e.From,
			Subject:      e.Subject,
			BodyHTML:     e.BodyHTML,
			BodyText:     e.BodyText,
			TemplateID:   e.TemplateID,
			TemplateData: e.TemplateData,
			Priority:     domain.EmailPriority(e.Priority),
		}
	}

	ucReq := emailuc.SendBulkEmailRequest{
		Emails:      emails,
		BatchSize:   req.BatchSize,
		StopOnError: req.StopOnError,
		RateLimit:   req.RateLimit,
		DryRun:      req.DryRun,
	}

	// Execute use case
	resp, err := s.useCases.SendBulkEmail.Execute(c.Request.Context(), ucReq)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	// Build response
	results := make([]BulkEmailResultResponse, len(resp.Results))
	for i, r := range resp.Results {
		errMsg := ""
		if r.Error != nil {
			errMsg = r.Error.Error()
		}

		results[i] = BulkEmailResultResponse{
			Index:     r.Index,
			EmailID:   r.EmailID,
			Success:   r.Success,
			MessageID: r.MessageID,
			Error:     errMsg,
			Timestamp: r.Timestamp,
		}
	}

	c.JSON(http.StatusOK, SendBulkEmailResponse{
		TotalEmails:  resp.TotalEmails,
		SuccessCount: resp.SuccessCount,
		FailureCount: resp.FailureCount,
		Results:      results,
		Duration:     resp.Duration.String(),
	})
}

// scheduleEmail handles POST /api/v1/emails/scheduled
func (s *Server) scheduleEmail(c *gin.Context) {
	var req ScheduleEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Build use case request
	ucReq := emailuc.ScheduleEmailRequest{
		Email: emailuc.SendEmailRequest{
			To:           req.Email.To,
			CC:           req.Email.CC,
			BCC:          req.Email.BCC,
			From:         req.Email.From,
			Subject:      req.Email.Subject,
			BodyHTML:     req.Email.BodyHTML,
			BodyText:     req.Email.BodyText,
			TemplateID:   req.Email.TemplateID,
			TemplateData: req.Email.TemplateData,
			Priority:     domain.EmailPriority(req.Email.Priority),
		},
		ScheduledAt: req.ScheduledAt,
	}

	// Execute use case
	resp, err := s.useCases.ScheduleEmail.Execute(c.Request.Context(), ucReq)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusCreated, ScheduleEmailResponse{
		EmailID:     resp.EmailID,
		Status:      string(resp.Status),
		ScheduledAt: resp.ScheduledAt,
		Message:     resp.Message,
	})
}

// getEmail handles GET /api/v1/emails/:id
func (s *Server) getEmail(c *gin.Context) {
	id := c.Param("id")

	resp, err := s.useCases.GetEmail.GetByID(c.Request.Context(), id)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, EmailDetailResponse{
		ID:                resp.ID,
		To:                resp.To,
		CC:                resp.CC,
		BCC:               resp.BCC,
		From:              resp.From,
		Subject:           resp.Subject,
		Status:            string(resp.Status),
		Priority:          string(resp.Priority),
		ProviderName:      resp.ProviderName,
		ProviderMessageID: resp.ProviderMessageID,
		RetryCount:        resp.RetryCount,
		MaxRetries:        resp.MaxRetries,
		LastError:         resp.LastError,
		CreatedAt:         resp.CreatedAt,
		UpdatedAt:         resp.UpdatedAt,
		SentAt:            resp.SentAt,
		ScheduledAt:       resp.ScheduledAt,
	})
}

// listEmails handles GET /api/v1/emails
func (s *Server) listEmails(c *gin.Context) {
	// Parse query parameters
	var query ListEmailsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		s.respondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// Set defaults
	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit < 1 {
		query.Limit = 50
	}
	if query.Limit > 100 {
		query.Limit = 100
	}

	// Build use case request
	ucReq := emailuc.ListEmailsRequest{
		Filters: domain.EmailFilters{
			Status: domain.EmailStatus(query.Status),
		},
		QueryOptions: domain.QueryOptions{
			Page:     query.Page,
			Limit:    query.Limit,
			SortBy:   query.SortBy,
			SortDesc: query.SortDesc,
		},
	}

	resp, err := s.useCases.GetEmail.ListEmails(c.Request.Context(), ucReq)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	// Build response
	emails := make([]EmailSummaryResponse, len(resp.Emails))
	for i, e := range resp.Emails {
		emails[i] = EmailSummaryResponse{
			ID:           e.ID,
			To:           e.To,
			Subject:      e.Subject,
			Status:       string(e.Status),
			Priority:     string(e.Priority),
			ProviderName: e.ProviderName,
			CreatedAt:    e.CreatedAt,
			SentAt:       e.SentAt,
		}
	}

	c.JSON(http.StatusOK, ListEmailsResponse{
		Emails:     emails,
		Total:      resp.Total,
		Page:       resp.Page,
		Limit:      resp.Limit,
		TotalPages: resp.TotalPages,
	})
}

// cancelScheduledEmail handles DELETE /api/v1/emails/:id/cancel
func (s *Server) cancelScheduledEmail(c *gin.Context) {
	id := c.Param("id")

	err := s.useCases.ScheduleEmail.CancelScheduledEmail(c.Request.Context(), id)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Email cancelled successfully",
	})
}