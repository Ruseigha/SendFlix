package httpapi

import (
	"net/http"

	"github.com/Ruseigha/SendFlix/internal/domain"
	"github.com/gin-gonic/gin"

	templateuc "github.com/Ruseigha/SendFlix/internal/usecase/template"
)

// createTemplate handles POST /api/v1/templates
func (s *Server) createTemplate(c *gin.Context) {
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Build use case request
	ucReq := templateuc.CreateTemplateRequest{
		Name:              req.Name,
		Description:       req.Description,
		Type:              domain.TemplateType(req.Type),
		Subject:           req.Subject,
		BodyHTML:          req.BodyHTML,
		BodyText:          req.BodyText,
		Language:          req.Language,
		Category:          req.Category,
		Tags:              req.Tags,
		RequiredVariables: req.RequiredVariables,
		OptionalVariables: req.OptionalVariables,
		DefaultVariables:  req.DefaultVariables,
		SampleData:        req.SampleData,
		CreatedBy:         "api", // TODO: Get from auth context
	}

	// Execute use case
	resp, err := s.useCases.CreateTemplate.Execute(c.Request.Context(), ucReq)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusCreated, CreateTemplateResponse{
		ID:        resp.ID,
		Name:      resp.Name,
		Status:    string(resp.Status),
		Version:   resp.Version,
		CreatedAt: resp.CreatedAt,
		Message:   resp.Message,
	})
}

// listTemplates handles GET /api/v1/templates
func (s *Server) listTemplates(c *gin.Context) {
	var query ListTemplatesQuery
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

	filters := domain.TemplateFilters{
		Status:   domain.TemplateStatus(query.Status),
		Category: query.Category,
		Language: query.Language,
		Search:   query.Search,
	}

	opts := domain.QueryOptions{
		Page:     query.Page,
		Limit:    query.Limit,
		SortBy:   query.SortBy,
		SortDesc: query.SortDesc,
	}

	resp, err := s.useCases.GetTemplate.ListTemplates(c.Request.Context(), filters, opts)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	// Build response
	templates := make([]TemplateSummaryResponse, len(resp.Templates))
	for i, t := range resp.Templates {
		templates[i] = TemplateSummaryResponse{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Type:        string(t.Type),
			Status:      string(t.Status),
			Version:     t.Version,
			Category:    t.Category,
			Language:    t.Language,
			UsageCount:  t.UsageCount,
			CreatedAt:   t.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, ListTemplatesResponse{
		Templates: templates,
	})
}

// getTemplate handles GET /api/v1/templates/:id
func (s *Server) getTemplate(c *gin.Context) {
	id := c.Param("id")

	resp, err := s.useCases.GetTemplate.GetByID(c.Request.Context(), id)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, TemplateDetailResponse{
		ID:                resp.ID,
		Name:              resp.Name,
		Description:       resp.Description,
		Type:              string(resp.Type),
		Status:            string(resp.Status),
		Version:           resp.Version,
		Subject:           resp.Subject,
		BodyHTML:          resp.BodyHTML,
		BodyText:          resp.BodyText,
		Language:          resp.Language,
		Category:          resp.Category,
		Tags:              resp.Tags,
		RequiredVariables: resp.RequiredVariables,
		OptionalVariables: resp.OptionalVariables,
		DefaultVariables:  resp.DefaultVariables,
		SampleData:        resp.SampleData,
		CreatedBy:         resp.CreatedBy,
		UpdatedBy:         resp.UpdatedBy,
		CreatedAt:         resp.CreatedAt,
		UpdatedAt:         resp.UpdatedAt,
		UsageCount:        resp.UsageCount,
		LastUsedAt:        resp.LastUsedAt,
	})
}

// updateTemplate handles PUT /api/v1/templates/:id
func (s *Server) updateTemplate(c *gin.Context) {
	id := c.Param("id")

	var req UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// Build use case request
	ucReq := templateuc.UpdateTemplateRequest{
		ID:          id,
		UpdatedBy:   "api", // TODO: Get from auth context
		ForceUpdate: req.ForceUpdate,
	}

	// Only set fields that are provided
	if req.Name != nil {
		ucReq.Name = req.Name
	}
	if req.Description != nil {
		ucReq.Description = req.Description
	}
	if req.Subject != nil {
		ucReq.Subject = req.Subject
	}
	if req.BodyHTML != nil {
		ucReq.BodyHTML = req.BodyHTML
	}
	if req.BodyText != nil {
		ucReq.BodyText = req.BodyText
	}

	// Execute use case
	resp, err := s.useCases.UpdateTemplate.Execute(c.Request.Context(), ucReq)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, UpdateTemplateResponse{
		ID:        resp.ID,
		Name:      resp.Name,
		Status:    string(resp.Status),
		Version:   resp.Version,
		UpdatedAt: resp.UpdatedAt,
		Message:   resp.Message,
		WasActive: resp.WasActive,
	})
}

// deleteTemplate handles DELETE /api/v1/templates/:id
func (s *Server) deleteTemplate(c *gin.Context) {
	id := c.Param("id")

	var query DeleteTemplateQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		s.respondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	ucReq := templateuc.DeleteTemplateRequest{
		ID:         id,
		DeletedBy:  "api", // TODO: Get from auth context
		HardDelete: query.HardDelete,
		Force:      query.Force,
	}

	resp, err := s.useCases.DeleteTemplate.Execute(c.Request.Context(), ucReq)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, DeleteTemplateResponse{
		ID:         resp.ID,
		Name:       resp.Name,
		Deleted:    resp.Deleted,
		Archived:   resp.Archived,
		Message:    resp.Message,
		EmailCount: resp.EmailCount,
	})
}

// activateTemplate handles POST /api/v1/templates/:id/activate
func (s *Server) activateTemplate(c *gin.Context) {
	id := c.Param("id")

	ucReq := templateuc.ActivateTemplateRequest{
		ID:          id,
		ActivatedBy: "api", // TODO: Get from auth context
	}

	resp, err := s.useCases.ActivateTemplate.Execute(c.Request.Context(), ucReq)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, ActivateTemplateResponse{
		ID:          resp.ID,
		Name:        resp.Name,
		Status:      string(resp.Status),
		Version:     resp.Version,
		ActivatedAt: resp.ActivatedAt,
		Message:     resp.Message,
	})
}

// deactivateTemplate handles POST /api/v1/templates/:id/deactivate
func (s *Server) deactivateTemplate(c *gin.Context) {
	id := c.Param("id")

	err := s.useCases.ActivateTemplate.DeactivateTemplate(
		c.Request.Context(),
		id,
		"api", // TODO: Get from auth context
	)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Template deactivated successfully",
	})
}

// previewTemplate handles POST /api/v1/templates/:id/preview
func (s *Server) previewTemplate(c *gin.Context) {
	id := c.Param("id")

	var req PreviewTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	ucReq := templateuc.PreviewTemplateRequest{
		TemplateID: id,
		Data:       req.Data,
		UseSample:  req.UseSample,
	}

	resp, err := s.useCases.PreviewTemplate.Execute(c.Request.Context(), ucReq)
	if err != nil {
		s.handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, PreviewTemplateResponse{
		TemplateID:   resp.TemplateID,
		TemplateName: resp.TemplateName,
		Subject:      resp.Subject,
		BodyHTML:     resp.BodyHTML,
		BodyText:     resp.BodyText,
		DataUsed:     resp.DataUsed,
		Warnings:     resp.Warnings,
	})
}