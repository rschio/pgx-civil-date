package date

import (
	"fmt"
	"reflect"
	"time"

	"cloud.google.com/go/civil"
	"github.com/jackc/pgx/v5/pgtype"
)

type Date civil.Date

func (d *Date) ScanDate(v pgtype.Date) error {
	if !v.Valid {
		*d = Date(civil.Date{})
		return nil
	}

	*d = Date(civil.DateOf(v.Time))
	return nil
}

func (d Date) DateValue() (pgtype.Date, error) {
	dd := civil.Date(d)
	return pgtype.Date{
		Time:  dd.In(time.UTC),
		Valid: dd.IsValid(),
	}, nil
}

func TryWrapDateEncodePlan(value any) (plan pgtype.WrappedEncodePlanNextSetter, nextValue any, ok bool) {
	switch value := value.(type) {
	case civil.Date:
		return &wrapDateEncodePlan{}, Date(value), true
	}

	return nil, nil, false
}

type wrapDateEncodePlan struct {
	next pgtype.EncodePlan
}

func (plan *wrapDateEncodePlan) SetNext(next pgtype.EncodePlan) { plan.next = next }

func (plan *wrapDateEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(Date(value.(civil.Date)), buf)
}

func TryWrapDateScanPlan(target any) (plan pgtype.WrappedScanPlanNextSetter, nextDst any, ok bool) {
	switch target := target.(type) {
	case *civil.Date:
		return &wrapDateScanPlan{}, (*Date)(target), true
	}

	return nil, nil, false
}

type wrapDateScanPlan struct {
	next pgtype.ScanPlan
}

func (plan *wrapDateScanPlan) SetNext(next pgtype.ScanPlan) { plan.next = next }

func (plan *wrapDateScanPlan) Scan(src []byte, dst interface{}) error {
	return plan.next.Scan(src, (*Date)(dst.(*civil.Date)))
}

type DateCodec struct {
	pgtype.DateCodec
}

func (DateCodec) DecodeValue(tm *pgtype.Map, oid uint32, format int16, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var target civil.Date
	scanPlan := tm.PlanScan(oid, format, &target)
	if scanPlan == nil {
		return nil, fmt.Errorf("PlanScan did not find a plan for civil.Date")
	}

	err := scanPlan.Scan(src, &target)
	if err != nil {
		return nil, err
	}

	return target, nil
}

func Register(m *pgtype.Map) {
	m.TryWrapEncodePlanFuncs = append([]pgtype.TryWrapEncodePlanFunc{TryWrapDateEncodePlan}, m.TryWrapEncodePlanFuncs...)
	m.TryWrapScanPlanFuncs = append([]pgtype.TryWrapScanPlanFunc{TryWrapDateScanPlan}, m.TryWrapScanPlanFuncs...)

	m.RegisterType(&pgtype.Type{
		Name:  "civil.Date",
		OID:   pgtype.DateOID,
		Codec: DateCodec{},
	})

	registerDefaultPgTypeVariants := func(name, arrayName string, value any) {
		// T
		m.RegisterDefaultPgType(value, name)

		// *T
		valueType := reflect.TypeOf(value)
		m.RegisterDefaultPgType(reflect.New(valueType).Interface(), name)

		// []T
		sliceType := reflect.SliceOf(valueType)
		m.RegisterDefaultPgType(reflect.MakeSlice(sliceType, 0, 0).Interface(), arrayName)

		// *[]T
		m.RegisterDefaultPgType(reflect.New(sliceType).Interface(), arrayName)

		// []*T
		sliceOfPointerType := reflect.SliceOf(reflect.TypeOf(reflect.New(valueType).Interface()))
		m.RegisterDefaultPgType(reflect.MakeSlice(sliceOfPointerType, 0, 0).Interface(), arrayName)

		// *[]*T
		m.RegisterDefaultPgType(reflect.New(sliceOfPointerType).Interface(), arrayName)
	}

	registerDefaultPgTypeVariants("civil.Date", "_civil.Date", civil.Date{})
	registerDefaultPgTypeVariants("civil.Date", "_civil.Date", Date{})
}
