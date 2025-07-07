-- Add a composite index to improve performance for CheckNodePosition
CREATE INDEX idx_resource_nodes_position
  ON resource_nodes(chunk_x, chunk_z, local_x, local_z, is_active);
