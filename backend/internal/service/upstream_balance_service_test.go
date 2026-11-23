package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeUpstreamBalanceStore struct {
	accounts     map[int64]*Account
	balanceCalls map[int64]float64
	extraMerges  map[int64]map[string]any
	thresholdSet map[int64]*float64
}

func newFakeUpstreamBalanceStore() *fakeUpstreamBalanceStore {
	return &fakeUpstreamBalanceStore{
		accounts:     make(map[int64]*Account),
		balanceCalls: make(map[int64]float64),
		extraMerges:  make(map[int64]map[string]any),
		thresholdSet: make(map[int64]*float64),
	}
}

func (f *fakeUpstreamBalanceStore) GetByID(_ context.Context, id int64) (*Account, error) {
	a, ok := f.accounts[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return a, nil
}

func (f *fakeUpstreamBalanceStore) ListActive(_ context.Context) ([]Account, error) {
	out := make([]Account, 0, len(f.accounts))
	for _, a := range f.accounts {
		out = append(out, *a)
	}
	return out, nil
}

func (f *fakeUpstreamBalanceStore) Update(_ context.Context, _ *Account) error { return nil }

func (f *fakeUpstreamBalanceStore) MergeAccountExtra(_ context.Context, id int64, patch map[string]any) error {
	f.extraMerges[id] = patch
	return nil
}

func (f *fakeUpstreamBalanceStore) UpdateUpstreamBalance(_ context.Context, id int64, balance float64, _ time.Time) error {
	f.balanceCalls[id] = balance
	return nil
}

func (f *fakeUpstreamBalanceStore) UpdateBalanceAlertThreshold(_ context.Context, id int64, threshold *float64) error {
	f.thresholdSet[id] = threshold
	return nil
}

func (f *fakeUpstreamBalanceStore) ListAccountsWithAlertThreshold(_ context.Context) ([]Account, error) {
	return nil, nil
}

func kiroBalanceTestAccount(id int64, balance float64) *Account {
	return &Account{
		ID:       id,
		Name:     fmt.Sprintf("acc-%d", id),
		Platform: PlatformKiro,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Credentials: map[string]any{
			"base_url": "https://upstream.example",
			"api_key":  "sk-test",
		},
		UpstreamBalance: &balance,
	}
}

func TestUpstreamBalanceService_CheckAlert(t *testing.T) {
	emails := []string{"ops@example.com"}
	cfg := UpstreamBalanceConfig{AlertEnabled: true, AlertThreshold: 10, AlertEmails: emails}

	t.Run("first drop below threshold dispatches once and marks active", func(t *testing.T) {
		store := newFakeUpstreamBalanceStore()
		store.accounts[1] = kiroBalanceTestAccount(1, 8.5)
		s := NewUpstreamBalanceService(store, nil, nil, nil)

		s.checkAlert(context.Background(), store.accounts[1], 8.5, cfg)

		if !store.extraMerges[1][extraKeyBalanceAlertActive].(bool) {
			t.Fatal("expected alert active flag set")
		}
	})

	t.Run("still below threshold does not re-dispatch", func(t *testing.T) {
		store := newFakeUpstreamBalanceStore()
		acc := kiroBalanceTestAccount(1, 8.5)
		acc.Extra = map[string]any{extraKeyBalanceAlertActive: true}
		store.accounts[1] = acc
		s := NewUpstreamBalanceService(store, nil, nil, nil)

		s.checkAlert(context.Background(), acc, 8.5, cfg)

		if _, dispatched := store.extraMerges[1]; dispatched {
			t.Fatal("expected no re-dispatch while already alerting")
		}
	})

	t.Run("recovery above threshold resets flag", func(t *testing.T) {
		store := newFakeUpstreamBalanceStore()
		acc := kiroBalanceTestAccount(1, 12)
		acc.Extra = map[string]any{extraKeyBalanceAlertActive: true}
		store.accounts[1] = acc
		s := NewUpstreamBalanceService(store, nil, nil, nil)

		s.checkAlert(context.Background(), acc, 12, cfg)

		if v, ok := store.extraMerges[1][extraKeyBalanceAlertActive].(bool); !ok || v {
			t.Fatal("expected alert flag reset to false on recovery")
		}
	})

	t.Run("per-account threshold overrides global", func(t *testing.T) {
		store := newFakeUpstreamBalanceStore()
		th := 5.0
		acc := kiroBalanceTestAccount(1, 8.5)
		acc.BalanceAlertThreshold = &th
		store.accounts[1] = acc
		s := NewUpstreamBalanceService(store, nil, nil, nil)

		// 全局 10、余额 8.5：若用全局会告警；但专属阈值 5 → 8.5 未跌破，不应告警。
		s.checkAlert(context.Background(), acc, 8.5, cfg)

		if _, dispatched := store.extraMerges[1]; dispatched {
			t.Fatal("expected per-account threshold to override global")
		}
	})

	t.Run("alert disabled skips check", func(t *testing.T) {
		store := newFakeUpstreamBalanceStore()
		acc := kiroBalanceTestAccount(1, 8.5)
		store.accounts[1] = acc
		s := NewUpstreamBalanceService(store, nil, nil, nil)

		s.checkAlert(context.Background(), acc, 8.5, UpstreamBalanceConfig{AlertEnabled: false, AlertThreshold: 10, AlertEmails: emails})

		if _, dispatched := store.extraMerges[1]; dispatched {
			t.Fatal("expected no dispatch when alerts disabled")
		}
	})
}

func TestUpstreamBalanceService_LoadConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("defaults when settings empty", func(t *testing.T) {
		repo := &fakeSettingRepo{vals: map[string]string{}}
		s := NewUpstreamBalanceService(newFakeUpstreamBalanceStore(), repo, nil, nil)
		cfg, err := s.LoadConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.SyncIntervalSeconds != DefaultUpstreamBalanceSyncIntervalSeconds {
			t.Fatalf("interval = %d, want default %d", cfg.SyncIntervalSeconds, DefaultUpstreamBalanceSyncIntervalSeconds)
		}
		if cfg.AlertThreshold != DefaultUpstreamBalanceAlertThreshold {
			t.Fatalf("threshold = %v, want default", cfg.AlertThreshold)
		}
		if !cfg.AlertEnabled {
			t.Fatal("expected alerts enabled by default")
		}
		if len(cfg.AlertEmails) != 1 || cfg.AlertEmails[0] != DefaultUpstreamBalanceAlertEmail {
			t.Fatalf("emails = %v, want default", cfg.AlertEmails)
		}
	})

	t.Run("configured values override defaults", func(t *testing.T) {
		repo := &fakeSettingRepo{vals: map[string]string{
			SettingKeyUpstreamBalanceSyncIntervalSeconds: "300",
			SettingKeyUpstreamBalanceAlertEnabled:        "false",
			SettingKeyUpstreamBalanceAlertEmails:         `["ops@example.com"]`,
			SettingKeyUpstreamBalanceAlertThreshold:      "7.5",
		}}
		s := NewUpstreamBalanceService(newFakeUpstreamBalanceStore(), repo, nil, nil)
		cfg, err := s.LoadConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.SyncIntervalSeconds != 300 || cfg.AlertEnabled || cfg.AlertThreshold != 7.5 {
			t.Fatalf("unexpected cfg: %+v", cfg)
		}
		if len(cfg.AlertEmails) != 1 || cfg.AlertEmails[0] != "ops@example.com" {
			t.Fatalf("emails = %v", cfg.AlertEmails)
		}
	})
}

func TestUpstreamBalanceService_FetchBalance(t *testing.T) {
	t.Run("parses balance from usage response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"balance": 12.345, "planName": "pro"}`)
		}))
		defer srv.Close()

		s := NewUpstreamBalanceService(newFakeUpstreamBalanceStore(), nil, nil, nil)
		balance, err := s.fetchBalance(context.Background(), srv.URL, "sk-key")
		if err != nil {
			t.Fatal(err)
		}
		if balance != 12.345 {
			t.Fatalf("balance = %v, want 12.345", balance)
		}
	})

	t.Run("missing balance field errors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `{"usage": {}}`)
		}))
		defer srv.Close()

		s := NewUpstreamBalanceService(newFakeUpstreamBalanceStore(), nil, nil, nil)
		if _, err := s.fetchBalance(context.Background(), srv.URL, "sk-key"); err == nil {
			t.Fatal("expected error when balance missing")
		}
	})

	t.Run("non-200 errors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer srv.Close()

		s := NewUpstreamBalanceService(newFakeUpstreamBalanceStore(), nil, nil, nil)
		if _, err := s.fetchBalance(context.Background(), srv.URL, "sk-key"); err == nil {
			t.Fatal("expected error on 403")
		}
	})
}

func TestUpstreamBalanceService_SaveConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("persists normalized config", func(t *testing.T) {
		repo := &fakeSettingRepo{vals: map[string]string{}}
		s := NewUpstreamBalanceService(newFakeUpstreamBalanceStore(), repo, nil, nil)
		err := s.SaveConfig(ctx, UpstreamBalanceConfig{
			SyncIntervalSeconds: 300,
			AlertEnabled:        true,
			AlertEmails:         []string{" a@x.com ", "b@x.com", ""},
			AlertThreshold:      8.5,
		})
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := s.LoadConfig(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.SyncIntervalSeconds != 300 || cfg.AlertThreshold != 8.5 || !cfg.AlertEnabled {
			t.Fatalf("unexpected cfg: %+v", cfg)
		}
		if len(cfg.AlertEmails) != 2 || cfg.AlertEmails[0] != "a@x.com" {
			t.Fatalf("emails not normalized: %v", cfg.AlertEmails)
		}
	})

	t.Run("rejects non-positive interval", func(t *testing.T) {
		s := NewUpstreamBalanceService(newFakeUpstreamBalanceStore(), &fakeSettingRepo{vals: map[string]string{}}, nil, nil)
		if err := s.SaveConfig(ctx, UpstreamBalanceConfig{SyncIntervalSeconds: 0}); err == nil {
			t.Fatal("expected error for zero interval")
		}
	})

	t.Run("rejects negative threshold", func(t *testing.T) {
		s := NewUpstreamBalanceService(newFakeUpstreamBalanceStore(), &fakeSettingRepo{vals: map[string]string{}}, nil, nil)
		if err := s.SaveConfig(ctx, UpstreamBalanceConfig{SyncIntervalSeconds: 60, AlertThreshold: -1}); err == nil {
			t.Fatal("expected error for negative threshold")
		}
	})
}
