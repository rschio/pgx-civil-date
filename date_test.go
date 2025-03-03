package date_test

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxtest"
	date "github.com/rschio/pgx-civil-date"
)

var defaultConnTestRunner pgxtest.ConnTestRunner

func init() {
	defaultConnTestRunner = pgxtest.DefaultConnTestRunner()
	defaultConnTestRunner.AfterConnect = func(ctx context.Context, t testing.TB, conn *pgx.Conn) {
		date.Register(conn.TypeMap())
	}
	defaultConnTestRunner.CreateConfig = func(ctx context.Context, t testing.TB) *pgx.ConnConfig {
		conn := "postgres://postgres:postgres@localhost/postgres?sslmode=disable"
		cfg, err := pgx.ParseConfig(conn)
		if err != nil {
			t.Fatal(err)
		}
		return cfg
	}
}

func TestCodecDecodeValue(t *testing.T) {
	defaultConnTestRunner.RunTest(t.Context(), t, func(ctx context.Context, t testing.TB, conn *pgx.Conn) {
		original := civil.Date{Year: 2025, Month: 2, Day: 28}

		rows, err := conn.Query(context.Background(), `select $1::date`, original)
		if err != nil {
			t.Fatal(err)
		}

		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				t.Fatal(err)
			}
			if len(values) != 1 {
				t.Fatalf("should have 1 value, got %d", len(values))
			}

			v0, ok := values[0].(civil.Date)
			if !ok || original.Compare(v0) != 0 {
				t.Fatal("should be equal")
			}
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}

		rows, err = conn.Query(context.Background(), `select $1::date`, nil)
		if err != nil {
			t.Fatal(err)
		}

		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				t.Fatal(err)
			}

			if len(values) != 1 {
				t.Fatalf("should have 1 value, got %d", len(values))
			}
			if values[0] != nil {
				t.Fatal("should be nil")
			}
		}

		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
	})
}

func isExpectedEq(a civil.Date) func(any) bool {
	return func(v any) bool {
		return a.Compare(v.(civil.Date)) == 0
	}
}

func TestDateRoundTrip(t *testing.T) {
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "date", []pgxtest.ValueRoundTripTest{
		{
			Param:  civil.Date{Year: 2025, Month: 2, Day: 28},
			Result: new(civil.Date),
			Test:   isExpectedEq(civil.Date{Year: 2025, Month: 2, Day: 28}),
		},
		{
			Param:  civil.Date{},
			Result: new(civil.Date),
			Test:   isExpectedEq(civil.DateOf(time.Time{})),
		},
	})
}
