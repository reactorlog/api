package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type stubScanner struct {
	values []any
	err    error
}

func (s stubScanner) Scan(dest ...any) error {
	if s.err != nil {
		return s.err
	}
	if len(dest) != len(s.values) {
		return errors.New("scan arity mismatch")
	}
	for i, value := range s.values {
		switch ptr := dest[i].(type) {
		case *string:
			*ptr = value.(string)
		case **string:
			if value == nil {
				*ptr = nil
			} else {
				text := value.(string)
				*ptr = &text
			}
		case **float64:
			if value == nil {
				*ptr = nil
			} else {
				number := value.(float64)
				*ptr = &number
			}
		case *time.Time:
			*ptr = value.(time.Time)
		case *int:
			*ptr = value.(int)
		default:
			return errors.New("unsupported scan destination")
		}
	}
	return nil
}

type stubRows struct {
	rows []stubScanner
	i    int
	err  error
}

func (r *stubRows) Next() bool {
	if r.i >= len(r.rows) {
		return false
	}
	r.i++
	return true
}

func (r *stubRows) Err() error                    { return r.err }
func (r *stubRows) Scan(dest ...any) error        { return r.rows[r.i-1].Scan(dest...) }
func (r *stubRows) Close()                        {}
func (r *stubRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *stubRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}
func (r *stubRows) Values() ([]any, error) { return nil, nil }
func (r *stubRows) RawValues() [][]byte    { return nil }
func (r *stubRows) Conn() *pgx.Conn        { return nil }
func (r *stubRows) TypeMap() *pgtype.Map   { return nil }

type stubDB struct {
	pingErr error
	query   func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	row     pgx.Row
}

func (d *stubDB) Ping(context.Context) error { return d.pingErr }

func (d *stubDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return d.query(ctx, sql, args...)
}

func (d *stubDB) QueryRow(context.Context, string, ...any) pgx.Row { return d.row }

func siteValues(now time.Time) []any {
	return []any{
		"11111111-1111-1111-1111-111111111111",
		nil, "Plant", nil, nil, nil, nil, nil, nil, nil, "Here", nil, nil, nil, now, now,
	}
}

func listSiteValues(now time.Time, reactorCount int) []any {
	return append(siteValues(now), reactorCount)
}

func reactorValues(now time.Time) []any {
	return []any{
		"22222222-2222-2222-2222-222222222222",
		"11111111-1111-1111-1111-111111111111",
		"Plant 1", "1", "05000001", "PWR", nil, nil, now, now,
	}
}

func TestValidUUID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{id: "11111111-1111-1111-1111-111111111111", want: true},
		{id: "11111111-1111-1111-1111-11111111111", want: false},
		{id: "11111111x1111-1111-1111-111111111111", want: false},
		{id: "11111111-1111-1111-1111-11111111111g", want: false},
		{id: "ABCDEFAB-CDEF-ABCD-ABCD-ABCDEFABCDEF", want: true},
	}
	for _, tt := range tests {
		if got := validUUID(tt.id); got != tt.want {
			t.Fatalf("validUUID(%q)=%v want %v", tt.id, got, tt.want)
		}
	}
}

func TestIsHex(t *testing.T) {
	for _, c := range "0123456789abcdefABCDEF" {
		if !isHex(c) {
			t.Fatalf("isHex(%q)=false", c)
		}
	}
	if isHex('g') || isHex('-') {
		t.Fatal("isHex accepted non-hex")
	}
}

func TestScanSiteAndReactor(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	site, err := scanSite(stubScanner{values: siteValues(now)})
	if err != nil || site.Name != "Plant" || site.Location == nil || *site.Location != "Here" {
		t.Fatalf("site=%+v err=%v", site, err)
	}
	if _, err := scanSite(stubScanner{err: errors.New("boom")}); err == nil {
		t.Fatal("expected scanSite error")
	}
	none, err := scanListSite(stubScanner{values: listSiteValues(now, 0)})
	if err != nil || none.Name != "Plant" || none.ReactorCount != 0 {
		t.Fatalf("list site=%+v err=%v", none, err)
	}
	counted, err := scanListSite(stubScanner{values: listSiteValues(now, 2)})
	if err != nil || counted.ReactorCount != 2 {
		t.Fatalf("list site=%+v err=%v", counted, err)
	}
	if _, err := scanListSite(stubScanner{err: errors.New("boom")}); err == nil {
		t.Fatal("expected scanListSite error")
	}

	reactor, err := scanReactor(stubScanner{values: reactorValues(now)})
	if err != nil || reactor.DocketNumber != "05000001" || reactor.CapacityMW != nil {
		t.Fatalf("reactor=%+v err=%v", reactor, err)
	}
	withMW := reactorValues(now)
	withMW[7] = 1138.5
	scanned, err := scanReactor(stubScanner{values: withMW})
	if err != nil || scanned.CapacityMW == nil || *scanned.CapacityMW != 1138.5 {
		t.Fatalf("reactor=%+v err=%v", scanned, err)
	}
	if _, err := scanReactor(stubScanner{err: errors.New("boom")}); err == nil {
		t.Fatal("expected scanReactor error")
	}
}

func TestCollectAndLoad(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	sites, err := collectSites(&stubRows{rows: []stubScanner{{values: listSiteValues(now, 0)}}})
	if err != nil || len(sites) != 1 || sites[0].ReactorCount != 0 {
		t.Fatalf("sites=%v err=%v", sites, err)
	}
	if _, err := collectSites(&stubRows{err: errors.New("rows")}); err == nil {
		t.Fatal("expected collectSites error")
	}

	reactors, err := collectReactors(&stubRows{rows: []stubScanner{{values: reactorValues(now)}}}, "list reactors")
	if err != nil || len(reactors) != 1 {
		t.Fatalf("reactors=%v err=%v", reactors, err)
	}
	if _, err := collectReactors(&stubRows{err: errors.New("rows")}, "list reactors"); err == nil {
		t.Fatal("expected collectReactors error")
	}

	site, err := loadSite(stubScanner{err: pgx.ErrNoRows})
	if !errors.Is(err, ErrNotFound) || site.ID != "" {
		t.Fatalf("loadSite miss: %+v %v", site, err)
	}
	if _, err := loadSite(stubScanner{err: errors.New("db")}); err == nil {
		t.Fatal("expected loadSite wrap")
	}
}

func TestPostgresStoreReads(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	id := "11111111-1111-1111-1111-111111111111"
	store := &PostgresStore{db: &stubDB{
		query: func(context.Context, string, ...any) (pgx.Rows, error) {
			return &stubRows{rows: []stubScanner{{values: listSiteValues(now, 2)}}}, nil
		},
		row: stubScanner{values: siteValues(now)},
	}}
	sites, err := store.ListSites(context.Background())
	if err != nil || len(sites) != 1 || sites[0].ReactorCount != 2 {
		t.Fatalf("ListSites=%v err=%v", sites, err)
	}

	store.db = &stubDB{query: func(context.Context, string, ...any) (pgx.Rows, error) {
		return nil, errors.New("query")
	}}
	if _, err := store.ListSites(context.Background()); err == nil {
		t.Fatal("expected ListSites error")
	}

	store.db = &stubDB{
		query: func(context.Context, string, ...any) (pgx.Rows, error) {
			return &stubRows{rows: []stubScanner{{values: reactorValues(now)}}}, nil
		},
		row: stubScanner{values: siteValues(now)},
	}
	reactors, err := store.ListReactors(context.Background())
	if err != nil || len(reactors) != 1 {
		t.Fatalf("ListReactors=%v err=%v", reactors, err)
	}
	if _, err := store.listReactorsForSite(context.Background(), id); err != nil {
		t.Fatal(err)
	}

	detail, err := store.SiteByID(context.Background(), id)
	if err != nil || detail.Name != "Plant" || detail.ReactorCount != len(detail.Reactors) || len(detail.Reactors) != 1 {
		t.Fatalf("detail=%+v err=%v", detail, err)
	}
	zeroStore := &PostgresStore{db: &stubDB{
		query: func(context.Context, string, ...any) (pgx.Rows, error) {
			return &stubRows{}, nil
		},
		row: stubScanner{values: siteValues(now)},
	}}
	zero, err := zeroStore.SiteByID(context.Background(), id)
	if err != nil || zero.ReactorCount != 0 || len(zero.Reactors) != 0 {
		t.Fatalf("zero detail=%+v err=%v", zero, err)
	}
	if _, err := store.SiteByID(context.Background(), "bad"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("bad id: %v", err)
	}

	store.db = &stubDB{row: stubScanner{err: pgx.ErrNoRows}, query: func(context.Context, string, ...any) (pgx.Rows, error) {
		return &stubRows{}, nil
	}}
	if _, err := store.SiteByID(context.Background(), id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing site: %v", err)
	}

	if err := (&PostgresStore{db: &stubDB{}}).Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}
