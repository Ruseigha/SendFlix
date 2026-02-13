-- Create templates table
CREATE TABLE IF NOT EXISTS templates (
    -- Identity
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    
    -- Type and status
    type VARCHAR(50) NOT NULL DEFAULT 'both',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    version INTEGER NOT NULL DEFAULT 1,
    
    -- Content
    subject TEXT NOT NULL,
    body_html TEXT,
    body_text TEXT,
    
    -- Metadata
    language VARCHAR(10) NOT NULL DEFAULT 'en',
    category VARCHAR(100),
    tags TEXT[] DEFAULT '{}',
    
    -- Variables
    required_variables TEXT[] DEFAULT '{}',
    optional_variables TEXT[] DEFAULT '{}',
    default_variables JSONB DEFAULT '{}',
    sample_data JSONB DEFAULT '{}',
    
    -- Audit
    created_by VARCHAR(255) NOT NULL,
    updated_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Usage statistics
    usage_count INTEGER NOT NULL DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT templates_type_check CHECK (type IN ('html', 'text', 'both')),
    CONSTRAINT templates_status_check CHECK (status IN ('draft', 'active', 'archived'))
);

-- Indexes
CREATE INDEX idx_templates_name ON templates(name);
CREATE INDEX idx_templates_status ON templates(status);
CREATE INDEX idx_templates_category ON templates(category);
CREATE INDEX idx_templates_language ON templates(language);
CREATE INDEX idx_templates_created_at ON templates(created_at DESC);
CREATE INDEX idx_templates_usage ON templates(usage_count DESC, last_used_at DESC);

-- GIN indexes
CREATE INDEX idx_templates_tags ON templates USING GIN(tags);
CREATE INDEX idx_templates_default_variables ON templates USING GIN(default_variables);
CREATE INDEX idx_templates_sample_data ON templates USING GIN(sample_data);

-- Full-text search index
CREATE INDEX idx_templates_search ON templates USING GIN(
    to_tsvector('english', coalesce(name, '') || ' ' || coalesce(description, ''))
);

-- Update timestamp trigger
CREATE TRIGGER update_templates_updated_at 
    BEFORE UPDATE ON templates
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments
COMMENT ON TABLE templates IS 'Email templates for reusable email content';
COMMENT ON COLUMN templates.status IS 'Template status: draft, active, or archived';
COMMENT ON COLUMN templates.version IS 'Template version, auto-incremented on update';
COMMENT ON COLUMN templates.usage_count IS 'Number of times this template has been used';