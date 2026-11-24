package service

import (
	"math"
	"testing"
)

func TestGroupLayerPriority(t *testing.T) {
	acc := Account{
		ID:       1,
		Priority: 50,
		AccountGroups: []AccountGroup{
			{AccountID: 1, GroupID: 3, Priority: 2},
			{AccountID: 1, GroupID: 5, Priority: 9},
		},
	}

	group3 := int64(3)
	if got := groupLayerPriority(acc, &group3); got != 2 {
		t.Fatalf("group 3 layer priority = %d, want 2", got)
	}

	group5 := int64(5)
	if got := groupLayerPriority(acc, &group5); got != 9 {
		t.Fatalf("group 5 layer priority = %d, want 9", got)
	}

	// 不属于该分组的候选降权到最后，避免账号级小值抢占分组内账号
	other := int64(7)
	if got := groupLayerPriority(acc, &other); got != unscheduledGroupLayerPriority {
		t.Fatalf("unrelated group layer priority = %d, want MaxInt32", got)
	}

	// 无分组调度回退账号级 priority（保持原语义）
	if got := groupLayerPriority(acc, nil); got != 50 {
		t.Fatalf("nil group layer priority = %d, want account-level 50", got)
	}
	zero := int64(0)
	if got := groupLayerPriority(acc, &zero); got != 50 {
		t.Fatalf("zero group layer priority = %d, want account-level 50", got)
	}
}

// 回归：调度候选入口替换分层键后，组内优先级（分组管理里编辑的）必须主导分层，
// 账号级 priority 与跨层账号都不得干扰；同层账号保持等值以参与负载均衡 + LRU。
func TestGroupLayerPriorityDrivesSchedulingLayer(t *testing.T) {
	groupID := int64(3)
	accounts := []Account{
		{ID: 1, Priority: 50, AccountGroups: []AccountGroup{{AccountID: 1, GroupID: 3, Priority: 1}}},
		{ID: 2, Priority: 10, AccountGroups: []AccountGroup{{AccountID: 2, GroupID: 3, Priority: 1}}},
		{ID: 3, Priority: 1, AccountGroups: []AccountGroup{{AccountID: 3, GroupID: 3, Priority: 2}}},
		{ID: 4, Priority: 7, AccountGroups: []AccountGroup{{AccountID: 4, GroupID: 9, Priority: 1}}},
	}

	// 模拟 SelectAccountWithLoadAwareness 候选入口的分层键替换
	for i := range accounts {
		accounts[i].Priority = groupLayerPriority(accounts[i], &groupID)
	}

	if accounts[0].Priority != 1 || accounts[1].Priority != 1 {
		t.Fatalf("same-layer accounts must share the group priority, got %d/%d", accounts[0].Priority, accounts[1].Priority)
	}
	if accounts[2].Priority != 2 {
		t.Fatalf("higher-layer account priority = %d, want 2", accounts[2].Priority)
	}
	if accounts[3].Priority != math.MaxInt32 {
		t.Fatalf("unrelated-group candidate must be deprioritized, got %d", accounts[3].Priority)
	}

	available := []accountWithLoad{
		{account: &accounts[0]},
		{account: &accounts[1]},
		{account: &accounts[2]},
		{account: &accounts[3]},
	}
	layer := filterByMinPriority(available)
	if len(layer) != 2 {
		t.Fatalf("layer size = %d, want 2 (both group priority=1 accounts)", len(layer))
	}
	for _, item := range layer {
		if item.account.ID == 3 {
			t.Fatal("account with higher group priority must not join the first layer")
		}
		if item.account.ID == 4 {
			t.Fatal("unrelated-group candidate must not join the first layer")
		}
	}
}
