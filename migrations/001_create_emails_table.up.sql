-- Create emails table
CREATE TABLE IF NOT EXISTS emails (
    -- Identity
    id VARCHAR(255) PRIMARY KEY,
    
    -- Recipients
    to_addresses TEXT[] NOT NULL,
    cc_addresses TEXT[] DEFAULT '{}',
    bcc_addresses TEXT[] DEFAULT '{}',
    
    -- Sender
    from_address VARCHAR(255) NOT NULL,
    reply_to VARCHAR(255),
    
    -- Content
    subject TEXT NOT NULL,
    body_html TEXT,
    body_text TEXT,
    
    -- Template
    template_id VARCHAR(255),
    template_data JSONB DEFAULT '{}',
    
    -- Metadata
    priority VARCHAR(50) NOT NULL DEFAULT 'normal',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    
    -- Provider
    provider_name VARCHAR(100),
    provider_message_id VARCHAR(255),
    
    -- Scheduling
    scheduled_at TIMESTAMP WITH TIME ZONE,
    
    -- Retry
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    last_error TEXT,
    
    -- Tracking
    track_opens BOOLEAN DEFAULT FALSE,
    track_clicks BOOLEAN DEFAULT FALSE,
    
    -- Additional metadata
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT emails_priority_check CHECK (priority IN ('low', 'normal', 'high')),
    CONSTRAINT emails_status_check CHECK (status IN ('pending', 'queued', 'sending', 'sent', 'failed', 'scheduled', 'cancelled', 'dead_letter')),
    CONSTRAINT emails_recipients_check CHECK (array_length(to_addresses, 1) > 0 OR array_length(cc_addresses, 1) > 0 OR array_length(bcc_addresses, 1) > 0)
);

-- Indexes for performance
CREATE INDEX idx_emails_status ON emails(status);
CREATE INDEX idx_emails_created_at ON emails(created_at DESC);
CREATE INDEX idx_emails_scheduled_at ON emails(scheduled_at) WHERE status = 'scheduled';
CREATE INDEX idx_emails_provider ON emails(provider_name);
CREATE INDEX idx_emails_from ON emails(from_address);

-- GIN index for JSONB fields
CREATE INDEX idx_emails_metadata ON emails USING GIN(metadata);
CREATE INDEX idx_emails_template_data ON emails USING GIN(template_data);

-- Composite index for retry queries
CREATE INDEX idx_emails_retry ON emails(status, retry_count) WHERE status = 'failed';

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_emails_updated_at 
    BEFORE UPDATE ON emails
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments
COMMENT ON TABLE emails IS 'Email messages and their delivery status';
COMMENT ON COLUMN emails.status IS 'Current delivery status of the email';
COMMENT ON COLUMN emails.retry_count IS 'Number of retry attempts made';
COMMENT ON COLUMN emails.metadata IS 'Additional custom metadata in JSON format';