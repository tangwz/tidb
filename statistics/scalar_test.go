// Copyright 2017 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package statistics

import (
	"math"

	. "github.com/pingcap/check"
	"github.com/pingcap/parser/mysql"
	"github.com/pingcap/tidb/types"
)

const eps = 1e-9

func getDecimal(value float64) *types.MyDecimal {
	dec := &types.MyDecimal{}
	dec.FromFloat64(value)
	return dec
}

func getDuration(value string) types.Duration {
	dur, _ := types.ParseDuration(nil, value, 0)
	return dur
}

func getTime(year, month, day int, timeType byte) types.Time {
	ret := types.NewTime(types.FromDate(year, month, day, 0, 0, 0, 0), timeType, types.DefaultFsp)
	return ret
}

func getTimeStamp(hour, min, sec int, timeType byte) types.Time {
	ret := types.NewTime(types.FromDate(2017, 1, 1, hour, min, sec, 0), timeType, 0)
	return ret
}

func getBinaryLiteral(value string) types.BinaryLiteral {
	b, _ := types.ParseBitStr(value)
	return b
}

func getUnsignedFieldType() *types.FieldType {
	tp := types.NewFieldType(mysql.TypeLonglong)
	tp.Flag |= mysql.UnsignedFlag
	return tp
}

func (s *testStatisticsSuite) TestCalcFraction(c *C) {
	tests := []struct {
		lower    types.Datum
		upper    types.Datum
		value    types.Datum
		fraction float64
		tp       *types.FieldType
	}{
		{
			lower:    types.NewIntDatum(0),
			upper:    types.NewIntDatum(4),
			value:    types.NewIntDatum(1),
			fraction: 0.25,
			tp:       types.NewFieldType(mysql.TypeLonglong),
		},
		{
			lower:    types.NewIntDatum(0),
			upper:    types.NewIntDatum(4),
			value:    types.NewIntDatum(4),
			fraction: 1,
			tp:       types.NewFieldType(mysql.TypeLonglong),
		},
		{
			lower:    types.NewIntDatum(0),
			upper:    types.NewIntDatum(4),
			value:    types.NewIntDatum(-1),
			fraction: 0,
			tp:       types.NewFieldType(mysql.TypeLonglong),
		},
		{
			lower:    types.NewUintDatum(0),
			upper:    types.NewUintDatum(4),
			value:    types.NewUintDatum(1),
			fraction: 0.25,
			tp:       getUnsignedFieldType(),
		},
		{
			lower:    types.NewFloat64Datum(0),
			upper:    types.NewFloat64Datum(4),
			value:    types.NewFloat64Datum(1),
			fraction: 0.25,
			tp:       types.NewFieldType(mysql.TypeDouble),
		},
		{
			lower:    types.NewFloat32Datum(0),
			upper:    types.NewFloat32Datum(4),
			value:    types.NewFloat32Datum(1),
			fraction: 0.25,
			tp:       types.NewFieldType(mysql.TypeFloat),
		},
		{
			lower:    types.NewDecimalDatum(getDecimal(0)),
			upper:    types.NewDecimalDatum(getDecimal(4)),
			value:    types.NewDecimalDatum(getDecimal(1)),
			fraction: 0.25,
			tp:       types.NewFieldType(mysql.TypeNewDecimal),
		},
		{
			lower:    types.NewMysqlBitDatum(getBinaryLiteral("0b0")),
			upper:    types.NewMysqlBitDatum(getBinaryLiteral("0b100")),
			value:    types.NewMysqlBitDatum(getBinaryLiteral("0b1")),
			fraction: 0.5,
			tp:       types.NewFieldType(mysql.TypeBit),
		},
		{
			lower:    types.NewDurationDatum(getDuration("0:00:00")),
			upper:    types.NewDurationDatum(getDuration("4:00:00")),
			value:    types.NewDurationDatum(getDuration("1:00:00")),
			fraction: 0.25,
			tp:       types.NewFieldType(mysql.TypeDuration),
		},
		{
			lower:    types.NewTimeDatum(getTime(2017, 1, 1, mysql.TypeTimestamp)),
			upper:    types.NewTimeDatum(getTime(2017, 4, 1, mysql.TypeTimestamp)),
			value:    types.NewTimeDatum(getTime(2017, 2, 1, mysql.TypeTimestamp)),
			fraction: 0.34444444444444444,
			tp:       types.NewFieldType(mysql.TypeTimestamp),
		},
		{
			lower:    types.NewTimeDatum(getTime(2017, 1, 1, mysql.TypeDatetime)),
			upper:    types.NewTimeDatum(getTime(2017, 4, 1, mysql.TypeDatetime)),
			value:    types.NewTimeDatum(getTime(2017, 2, 1, mysql.TypeDatetime)),
			fraction: 0.34444444444444444,
			tp:       types.NewFieldType(mysql.TypeDatetime),
		},
		{
			lower:    types.NewTimeDatum(getTime(2017, 1, 1, mysql.TypeDate)),
			upper:    types.NewTimeDatum(getTime(2017, 4, 1, mysql.TypeDate)),
			value:    types.NewTimeDatum(getTime(2017, 2, 1, mysql.TypeDate)),
			fraction: 0.34444444444444444,
			tp:       types.NewFieldType(mysql.TypeDate),
		},
		{
			lower:    types.NewStringDatum("aasad"),
			upper:    types.NewStringDatum("addad"),
			value:    types.NewStringDatum("abfsd"),
			fraction: 0.32280253984063745,
			tp:       types.NewFieldType(mysql.TypeString),
		},
		{
			lower:    types.NewBytesDatum([]byte("aasad")),
			upper:    types.NewBytesDatum([]byte("asdff")),
			value:    types.NewBytesDatum([]byte("abfsd")),
			fraction: 0.0529216802217269,
			tp:       types.NewFieldType(mysql.TypeBlob),
		},
	}
	for _, test := range tests {
		hg := NewHistogram(0, 0, 0, 0, test.tp, 1, 0)
		hg.AppendBucket(&test.lower, &test.upper, 0, 0)
		hg.PreCalculateScalar()
		fraction := hg.calcFraction(0, &test.value)
		c.Check(math.Abs(fraction-test.fraction) < eps, IsTrue)
	}
}

func (s *testStatisticsSuite) TestEnumRangeValues(c *C) {
	tests := []struct {
		low         types.Datum
		high        types.Datum
		lowExclude  bool
		highExclude bool
		res         string
	}{
		{
			low:         types.NewIntDatum(0),
			high:        types.NewIntDatum(5),
			lowExclude:  false,
			highExclude: true,
			res:         "(0, 1, 2, 3, 4)",
		},
		{
			low:         types.NewIntDatum(math.MinInt64),
			high:        types.NewIntDatum(math.MaxInt64),
			lowExclude:  false,
			highExclude: false,
			res:         "",
		},
		{
			low:         types.NewUintDatum(0),
			high:        types.NewUintDatum(5),
			lowExclude:  false,
			highExclude: true,
			res:         "(0, 1, 2, 3, 4)",
		},
		{
			low:         types.NewDurationDatum(getDuration("0:00:00")),
			high:        types.NewDurationDatum(getDuration("0:00:05")),
			lowExclude:  false,
			highExclude: true,
			res:         "(00:00:00, 00:00:01, 00:00:02, 00:00:03, 00:00:04)",
		},
		{
			low:         types.NewDurationDatum(getDuration("0:00:00")),
			high:        types.NewDurationDatum(getDuration("0:00:05")),
			lowExclude:  false,
			highExclude: true,
			res:         "(00:00:00, 00:00:01, 00:00:02, 00:00:03, 00:00:04)",
		},
		{
			low:         types.NewTimeDatum(getTime(2017, 1, 1, mysql.TypeDate)),
			high:        types.NewTimeDatum(getTime(2017, 1, 5, mysql.TypeDate)),
			lowExclude:  false,
			highExclude: true,
			res:         "(2017-01-01, 2017-01-02, 2017-01-03, 2017-01-04)",
		},
		{
			low:         types.NewTimeDatum(getTimeStamp(0, 0, 0, mysql.TypeTimestamp)),
			high:        types.NewTimeDatum(getTimeStamp(0, 0, 5, mysql.TypeTimestamp)),
			lowExclude:  false,
			highExclude: true,
			res:         "(2017-01-01 00:00:00, 2017-01-01 00:00:01, 2017-01-01 00:00:02, 2017-01-01 00:00:03, 2017-01-01 00:00:04)",
		},
		{
			low:         types.NewTimeDatum(getTimeStamp(0, 0, 0, mysql.TypeDatetime)),
			high:        types.NewTimeDatum(getTimeStamp(0, 0, 5, mysql.TypeDatetime)),
			lowExclude:  false,
			highExclude: true,
			res:         "(2017-01-01 00:00:00, 2017-01-01 00:00:01, 2017-01-01 00:00:02, 2017-01-01 00:00:03, 2017-01-01 00:00:04)",
		},
		// fix issue 11610
		{
			low:         types.NewIntDatum(math.MinInt64),
			high:        types.NewIntDatum(0),
			lowExclude:  false,
			highExclude: false,
			res:         "",
		},
	}
	for _, t := range tests {
		vals := enumRangeValues(t.low, t.high, t.lowExclude, t.highExclude)
		str, err := types.DatumsToString(vals, true)
		c.Assert(err, IsNil)
		c.Assert(str, Equals, t.res)
	}
}

func (s *testStatisticsSuite) TestCalcWidth4Datums(c *C) {
	tests := []struct {
		left  types.Datum
		right types.Datum
		lower types.Datum
		upper types.Datum
		width float64
		tp    *types.FieldType
	}{
		{
			left:  types.NewIntDatum(0),
			right: types.NewIntDatum(1),
			lower: types.NewIntDatum(0),
			upper: types.NewIntDatum(4),
			width: 1,
			tp:    types.NewFieldType(mysql.TypeLonglong),
		},
		{
			left:  types.NewIntDatum(0),
			right: types.NewIntDatum(4),
			lower: types.NewIntDatum(0),
			upper: types.NewIntDatum(4),
			width: 4,
			tp:    types.NewFieldType(mysql.TypeLonglong),
		},
		{
			left:  types.NewIntDatum(0),
			right: types.NewIntDatum(-1),
			lower: types.NewIntDatum(0),
			upper: types.NewIntDatum(4),
			width: 0,
			tp:    types.NewFieldType(mysql.TypeLonglong),
		},
		{
			left:  types.NewUintDatum(0),
			right: types.NewUintDatum(1),
			lower: types.NewUintDatum(0),
			upper: types.NewUintDatum(4),
			width: 1,
			tp:    getUnsignedFieldType(),
		},
		{
			left:  types.NewFloat64Datum(0),
			right: types.NewFloat64Datum(1),
			lower: types.NewFloat64Datum(0),
			upper: types.NewFloat64Datum(4),
			width: 1,
			tp:    types.NewFieldType(mysql.TypeDouble),
		},
		{
			left:  types.NewFloat32Datum(0),
			right: types.NewFloat32Datum(1),
			lower: types.NewFloat32Datum(0),
			upper: types.NewFloat32Datum(4),
			width: 1,
			tp:    types.NewFieldType(mysql.TypeFloat),
		},
		{
			left:  types.NewDecimalDatum(getDecimal(0)),
			right: types.NewDecimalDatum(getDecimal(1)),
			lower: types.NewDecimalDatum(getDecimal(0)),
			upper: types.NewDecimalDatum(getDecimal(4)),
			width: 1,
			tp:    types.NewFieldType(mysql.TypeNewDecimal),
		},
		{
			left:  types.NewMysqlBitDatum(getBinaryLiteral("0b0")),
			right: types.NewMysqlBitDatum(getBinaryLiteral("0b1")),
			lower: types.NewMysqlBitDatum(getBinaryLiteral("0b0")),
			upper: types.NewMysqlBitDatum(getBinaryLiteral("0b100")),
			width: 0,
			tp:    types.NewFieldType(mysql.TypeBit),
		},
		{
			left:  types.NewDurationDatum(getDuration("0:00:00")),
			right: types.NewDurationDatum(getDuration("1:00:00")),
			lower: types.NewDurationDatum(getDuration("0:00:00")),
			upper: types.NewDurationDatum(getDuration("4:00:00")),
			width: 3600000000000,
			tp:    types.NewFieldType(mysql.TypeDuration),
		},
		{
			left:  types.NewTimeDatum(getTime(2017, 1, 1, mysql.TypeTimestamp)),
			right: types.NewTimeDatum(getTime(2017, 2, 1, mysql.TypeTimestamp)),
			lower: types.NewTimeDatum(getTime(2017, 1, 1, mysql.TypeTimestamp)),
			upper: types.NewTimeDatum(getTime(2017, 4, 1, mysql.TypeTimestamp)),
			width: 2678400000000000,
			tp:    types.NewFieldType(mysql.TypeTimestamp),
		},
		{
			left:  types.NewTimeDatum(getTime(2017, 1, 1, mysql.TypeDatetime)),
			right: types.NewTimeDatum(getTime(2017, 2, 1, mysql.TypeDatetime)),
			lower: types.NewTimeDatum(getTime(2017, 1, 1, mysql.TypeDatetime)),
			upper: types.NewTimeDatum(getTime(2017, 4, 1, mysql.TypeDatetime)),
			width: 2678400000000000,
			tp:    types.NewFieldType(mysql.TypeDatetime),
		},
		{
			left:  types.NewTimeDatum(getTime(2017, 1, 1, mysql.TypeDate)),
			right: types.NewTimeDatum(getTime(2017, 2, 1, mysql.TypeDate)),
			lower: types.NewTimeDatum(getTime(2017, 1, 1, mysql.TypeDate)),
			upper: types.NewTimeDatum(getTime(2017, 4, 1, mysql.TypeDate)),
			width: 2678400000000000,
			tp:    types.NewFieldType(mysql.TypeDate),
		},
		{
			left:  types.NewStringDatum("aasad"),
			right: types.NewStringDatum("abfsd"),
			lower: types.NewStringDatum("aasad"),
			upper: types.NewStringDatum("addad"),
			width: 68418210549989376,
			tp:    types.NewFieldType(mysql.TypeString),
		},
		{
			left:  types.NewBytesDatum([]byte("aasad")),
			right: types.NewBytesDatum([]byte("abfsd")),
			lower: types.NewBytesDatum([]byte("aasad")),
			upper: types.NewBytesDatum([]byte("asdff")),
			width: 68418210549989376,
			tp:    types.NewFieldType(mysql.TypeBlob),
		},
	}
	for _, test := range tests {
		hg := NewHistogram(0, 0, 0, 0, test.tp, 1, 0)
		hg.AppendBucket(&test.lower, &test.upper, 0, 0)
		hg.PreCalculateScalar()
		width := calcWidth4Datums(&test.left, &test.right, &test.lower, &test.upper)
		c.Check(math.Abs(width-test.width) < eps, IsTrue)
	}
}
