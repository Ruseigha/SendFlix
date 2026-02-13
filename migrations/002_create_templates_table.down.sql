-- Drop templates table and related objects
DROP TRIGGER IF EXISTS update_templates_updated_at ON templates;
DROP TABLE IF EXISTS templates;