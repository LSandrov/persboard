INSERT INTO eazybi_metric_weights (metric_key, title, weight)
VALUES
  ('tickets', 'Tickets', 1.0000),
  ('storyPoints', 'Story Points', 0.5000),
  ('defectsPerSprint', 'Defects / sprint', 1.0000)
ON CONFLICT (metric_key) DO NOTHING;

