package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupUpstreamGroupLinksMigration(t *testing.T) {
	content, err := FS.ReadFile("254_group_upstream_group_links.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	// 建表与两列外键：本站分组、上游分组都必须级联删除，避免留下孤儿关联。
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS group_upstream_group_links")
	require.Contains(t, sql, "group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE")
	require.Contains(t, sql, "upstream_group_id BIGINT NOT NULL REFERENCES upstream_groups(id) ON DELETE CASCADE")

	// 同一对关联只保留一条有效记录；软删后的历史行不参与唯一约束。
	require.Contains(t, sql,
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_group_upstream_group_links_unique_active ON group_upstream_group_links (group_id, upstream_group_id) WHERE deleted_at IS NULL")

	// 双向索引：按本站分组查供货关系，按上游分组查被哪些分组使用。
	require.Contains(t, sql, "idx_group_upstream_group_links_group")
	require.Contains(t, sql, "idx_group_upstream_group_links_upstream_group")

	require.Contains(t, sql, "COMMENT ON TABLE group_upstream_group_links")
}
