-- Sample data for testing the Agents page
-- Run this after the initial migration

-- Insert sample projects (idempotent)
INSERT INTO projects (id, name, description, status, created_at) VALUES
('550e8400-e29b-41d4-a716-446655440001', 'E-Commerce Platform', 'Building a modern e-commerce solution', 'active', NOW()),
('550e8400-e29b-41d4-a716-446655440002', 'Mobile App Development', 'Cross-platform mobile application', 'active', NOW()),
('550e8400-e29b-41d4-a716-446655440003', 'Data Analytics Dashboard', 'Real-time analytics and reporting', 'inactive', NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert sample agents (idempotent)
INSERT INTO agents (id, project_id, name, role, team, status, last_seen, created_at) VALUES
-- E-Commerce Platform agents
('660e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', 'React Frontend Agent', 'frontend', 'UI Team', 'active', NOW(), NOW() - INTERVAL '1 day'),
('660e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', 'Node Backend Agent', 'backend', 'API Team', 'active', NOW(), NOW() - INTERVAL '1 day'),
('660e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440001', 'Payment Integration Agent', 'backend', 'Integration Team', 'active', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 days'),

-- Mobile App Development agents
('660e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440002', 'Flutter UI Agent', 'frontend', 'Mobile Team', 'active', NOW(), NOW() - INTERVAL '12 hours'),
('660e8400-e29b-41d4-a716-446655440005', '550e8400-e29b-41d4-a716-446655440002', 'API Integration Agent', 'backend', 'Mobile Team', 'active', NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '12 hours'),
('660e8400-e29b-41d4-a716-446655440006', '550e8400-e29b-41d4-a716-446655440002', 'Firebase Agent', 'backend', 'Cloud Team', 'inactive', NOW() - INTERVAL '5 days', NOW() - INTERVAL '15 days'),

-- Data Analytics Dashboard agents
('660e8400-e29b-41d4-a716-446655440007', '550e8400-e29b-41d4-a716-446655440003', 'Dashboard Frontend Agent', 'frontend', 'Analytics Team', 'active', NOW() - INTERVAL '1 hour', NOW() - INTERVAL '3 days'),
('660e8400-e29b-41d4-a716-446655440008', '550e8400-e29b-41d4-a716-446655440003', 'Data Processing Agent', 'backend', 'Analytics Team', 'active', NOW(), NOW() - INTERVAL '3 days'),
('660e8400-e29b-41d4-a716-446655440009', '550e8400-e29b-41d4-a716-446655440003', 'Reporting Agent', 'backend', 'BI Team', 'inactive', NOW() - INTERVAL '10 days', NOW() - INTERVAL '30 days')
ON CONFLICT (id) DO NOTHING;

-- Insert sample tasks (idempotent)
INSERT INTO tasks (id, project_id, created_by, assigned_to, title, description, status, priority, created_at) VALUES
-- E-Commerce tasks
('770e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440001', 'Implement Product Listing Page', 'Create responsive product grid with filtering', 'in_progress', 'high', NOW() - INTERVAL '2 days'),
('770e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440002', '660e8400-e29b-41d4-a716-446655440002', 'Setup Shopping Cart API', 'RESTful API for cart operations', 'in_progress', 'high', NOW() - INTERVAL '2 days'),
('770e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440003', '660e8400-e29b-41d4-a716-446655440003', 'Integrate Stripe Payment', 'Complete payment gateway integration', 'pending', 'high', NOW() - INTERVAL '1 day'),

-- Mobile App tasks
('770e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440002', '660e8400-e29b-41d4-a716-446655440004', '660e8400-e29b-41d4-a716-446655440004', 'Design App Navigation', 'Bottom navigation with tabs', 'done', 'medium', NOW() - INTERVAL '5 days'),
('770e8400-e29b-41d4-a716-446655440005', '550e8400-e29b-41d4-a716-446655440002', '660e8400-e29b-41d4-a716-446655440005', '660e8400-e29b-41d4-a716-446655440005', 'Implement User Authentication', 'OAuth2 + JWT implementation', 'in_progress', 'high', NOW() - INTERVAL '3 days'),
('770e8400-e29b-41d4-a716-446655440006', '550e8400-e29b-41d4-a716-446655440002', '660e8400-e29b-41d4-a716-446655440006', '660e8400-e29b-41d4-a716-446655440006', 'Setup Push Notifications', 'Firebase Cloud Messaging setup', 'pending', 'medium', NOW() - INTERVAL '1 day'),

-- Analytics Dashboard tasks
('770e8400-e29b-41d4-a716-446655440007', '550e8400-e29b-41d4-a716-446655440003', '660e8400-e29b-41d4-a716-446655440007', '660e8400-e29b-41d4-a716-446655440007', 'Create Chart Components', 'Reusable chart library integration', 'done', 'high', NOW() - INTERVAL '10 days'),
('770e8400-e29b-41d4-a716-446655440008', '550e8400-e29b-41d4-a716-446655440003', '660e8400-e29b-41d4-a716-446655440008', '660e8400-e29b-41d4-a716-446655440008', 'Build ETL Pipeline', 'Data extraction and transformation', 'in_progress', 'high', NOW() - INTERVAL '7 days'),
('770e8400-e29b-41d4-a716-446655440009', '550e8400-e29b-41d4-a716-446655440003', '660e8400-e29b-41d4-a716-446655440009', '660e8400-e29b-41d4-a716-446655440009', 'Generate PDF Reports', 'Export functionality for reports', 'blocked', 'low', NOW() - INTERVAL '5 days')
ON CONFLICT (id) DO NOTHING;

-- Insert sample contexts (idempotent)
INSERT INTO contexts (id, project_id, agent_id, task_id, title, content, tags, created_at, updated_at) VALUES
('880e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440001', '770e8400-e29b-41d4-a716-446655440001', 'Product Listing API Documentation', '# API Endpoints\n\n## GET /api/products\nReturns list of products with pagination\n\n### Query Parameters\n- page: int (default: 1)\n- limit: int (default: 20)\n- category: string (optional)', ARRAY['api', 'documentation', 'products'], NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
('880e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440002', '770e8400-e29b-41d4-a716-446655440002', 'Cart Data Model', '# Shopping Cart Schema\n\n```json\n{\n  "id": "uuid",\n  "user_id": "uuid",\n  "items": [\n    {\n      "product_id": "uuid",\n      "quantity": 1,\n      "price": 29.99\n    }\n  ],\n  "total": 29.99\n}\n```', ARRAY['database', 'schema', 'cart'], NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
('880e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440002', '660e8400-e29b-41d4-a716-446655440004', '770e8400-e29b-41d4-a716-446655440004', 'Flutter Navigation Setup', '# Navigation Implementation\n\nUsing `go_router` package for declarative routing.\n\n## Routes\n- / - Home\n- /profile - User Profile\n- /settings - App Settings', ARRAY['flutter', 'navigation', 'mobile'], NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),
('880e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440003', '660e8400-e29b-41d4-a716-446655440007', '770e8400-e29b-41d4-a716-446655440007', 'Chart.js Integration Guide', '# Using Chart.js with React\n\n```javascript\nimport { Line } from ''react-chartjs-2'';\n\nconst MyChart = () => (\n  <Line data={data} options={options} />\n);\n```', ARRAY['charts', 'react', 'visualization'], NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days')
ON CONFLICT (id) DO NOTHING;
