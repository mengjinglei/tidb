// Copyright 2015 PingCAP, Inc.
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

package expression

import (
	"math"
	"math/rand"
	"runtime"
	"time"

	. "github.com/pingcap/check"
	"github.com/pingcap/tidb/ast"
	"github.com/pingcap/tidb/mysql"
	"github.com/pingcap/tidb/util/testleak"
	"github.com/pingcap/tidb/util/testutil"
	"github.com/pingcap/tidb/util/types"
)

func (s *testEvaluatorSuite) TestAbs(c *C) {
	defer testleak.AfterTest(c)()
	tbl := []struct {
		Arg interface{}
		Ret interface{}
	}{
		{nil, nil},
		{int64(1), int64(1)},
		{uint64(1), uint64(1)},
		{int64(-1), int64(1)},
		{float64(3.14), float64(3.14)},
		{float64(-3.14), float64(3.14)},
	}

	Dtbl := tblToDtbl(tbl)

	for _, t := range Dtbl {
		fc := funcs[ast.Abs]
		f, err := fc.getFunction(datumsToConstants(t["Arg"]), s.ctx)
		c.Assert(err, IsNil)
		v, err := f.eval(nil)
		c.Assert(err, IsNil)
		c.Assert(v, testutil.DatumEquals, t["Ret"][0])
	}
}

func (s *testEvaluatorSuite) TestCeil(c *C) {
	defer testleak.AfterTest(c)()
	tbl := []struct {
		Arg interface{}
		Ret interface{}
	}{
		{nil, nil},
		{int64(1), int64(1)},
		{float64(1.23), float64(2)},
		{float64(-1.23), float64(-1)},
		{"1.23", float64(2)},
		{"-1.23", float64(-1)},
	}

	Dtbl := tblToDtbl(tbl)

	for _, t := range Dtbl {
		fc := funcs[ast.Ceil]
		f, err := fc.getFunction(datumsToConstants(t["Arg"]), s.ctx)
		c.Assert(err, IsNil)
		v, err := f.eval(nil)
		c.Assert(err, IsNil)
		c.Assert(v, DeepEquals, t["Ret"][0], Commentf("arg:%v", t["Arg"]))
	}
}

func (s *testEvaluatorSuite) TestExp(c *C) {
	defer testleak.AfterTest(c)()
	testcases := []struct {
		num interface{}
		ret interface{}
		err Checker
	}{
		{int64(1), float64(2.718281828459045), IsNil},
		{float64(1.23), float64(3.4212295362896734), IsNil},
		{float64(-1.23), float64(0.2922925776808594), IsNil},
		{float64(-1), float64(0.36787944117144233), IsNil},
		{float64(0), float64(1), IsNil},
		{"1.23", float64(3.4212295362896734), IsNil},
		{"-1.23", float64(0.2922925776808594), IsNil},
		{"0", float64(1), IsNil},
		{nil, nil, IsNil},
		{"abce", nil, NotNil},
		{"", nil, NotNil},
	}
	for _, t := range testcases {
		if runtime.GOARCH == "ppc64le" && t.num == int64(1) {
			t.ret = float64(2.7182818284590455)
		}
		fc := funcs[ast.Exp]
		f, err := fc.getFunction(datumsToConstants(types.MakeDatums(t.num)), s.ctx)
		c.Assert(err, IsNil)
		v, err := f.eval(nil)
		c.Assert(err, t.err)
		c.Assert(v, testutil.DatumEquals, types.NewDatum(t.ret))
	}
}

func (s *testEvaluatorSuite) TestFloor(c *C) {
	defer testleak.AfterTest(c)()

	sc := s.ctx.GetSessionVars().StmtCtx
	tmpIT := sc.IgnoreTruncate
	sc.IgnoreTruncate = true
	defer func() {
		sc.IgnoreTruncate = tmpIT
	}()

	genDuration := func(h, m, s int64) types.Duration {
		duration := time.Duration(h)*time.Hour +
			time.Duration(m)*time.Minute +
			time.Duration(s)*time.Second

		return types.Duration{Duration: duration, Fsp: types.DefaultFsp}
	}

	genTime := func(y, m, d int) types.Time {
		return types.Time{
			Time: types.FromDate(y, m, d, 0, 0, 0, 0),
			Type: mysql.TypeDatetime,
			Fsp:  types.DefaultFsp}
	}

	for _, test := range []struct {
		arg    interface{}
		expect interface{}
		isNil  bool
		getErr bool
	}{
		{nil, nil, true, false},
		{int64(1), int64(1), false, false},
		{float64(1.23), float64(1), false, false},
		{float64(-1.23), float64(-2), false, false},
		{"1.23", float64(1), false, false},
		{"-1.23", float64(-2), false, false},
		{"-1.b23", float64(-1), false, false},
		{"abce", float64(0), false, false},
		{genDuration(12, 59, 59), float64(125959), false, false},
		{genDuration(0, 12, 34), float64(1234), false, false},
		{genTime(2017, 7, 19), float64(20170719000000), false, false},
	} {
		f, err := newFunctionForTest(s.ctx, ast.Floor, primitiveValsToConstants([]interface{}{test.arg})...)
		c.Assert(err, IsNil)

		result, err := f.Eval(nil)
		if test.getErr {
			c.Assert(err, NotNil)
		} else {
			c.Assert(err, IsNil)
			if test.isNil {
				c.Assert(result.Kind(), Equals, types.KindNull)
			} else {
				c.Assert(result, testutil.DatumEquals, types.NewDatum(test.expect))
			}
		}
	}

	for _, exp := range []Expression{
		&Constant{
			Value:   types.NewDatum(0),
			RetType: types.NewFieldType(mysql.TypeTiny),
		},
		&Constant{
			Value:   types.NewFloat64Datum(float64(12.34)),
			RetType: types.NewFieldType(mysql.TypeFloat),
		},
	} {
		f, err := funcs[ast.Floor].getFunction([]Expression{exp}, s.ctx)
		c.Assert(err, IsNil)
		c.Assert(f.isDeterministic(), IsTrue)
	}
}

func (s *testEvaluatorSuite) TestLog(c *C) {
	defer testleak.AfterTest(c)()

	tests := []struct {
		args   []interface{}
		expect float64
		isNil  bool
		getErr bool
	}{
		{[]interface{}{nil}, 0, true, false},
		{[]interface{}{nil, nil}, 0, true, false},
		{[]interface{}{int64(100)}, 4.605170185988092, false, false},
		{[]interface{}{float64(100)}, 4.605170185988092, false, false},
		{[]interface{}{int64(10), int64(100)}, 2, false, false},
		{[]interface{}{float64(10), float64(100)}, 2, false, false},
		{[]interface{}{float64(-1)}, 0, true, false},
		{[]interface{}{float64(1), float64(2)}, 0, true, false},
		{[]interface{}{float64(0.5), float64(0.25)}, 2, false, false},
		{[]interface{}{"abc"}, 0, false, true},
	}

	for _, test := range tests {
		f, err := newFunctionForTest(s.ctx, ast.Log, primitiveValsToConstants(test.args)...)
		c.Assert(err, IsNil)

		result, err := f.Eval(nil)
		if test.getErr {
			c.Assert(err, NotNil)
		} else {
			c.Assert(err, IsNil)
			if test.isNil {
				c.Assert(result.Kind(), Equals, types.KindNull)
			} else {
				c.Assert(result.GetFloat64(), Equals, test.expect)
			}
		}
	}

	f, err := funcs[ast.Log].getFunction([]Expression{Zero}, s.ctx)
	c.Assert(err, IsNil)
	c.Assert(f.isDeterministic(), IsTrue)
}

func (s *testEvaluatorSuite) TestLog2(c *C) {
	defer testleak.AfterTest(c)()

	tests := []struct {
		args   interface{}
		expect float64
		isNil  bool
		getErr bool
	}{
		{nil, 0, true, false},
		{int64(16), 4, false, false},
		{float64(16), 4, false, false},
		{int64(5), 2.321928094887362, false, false},
		{int64(-1), 0, true, false},
		{"4abc", 0, false, true},
	}

	for _, test := range tests {
		f, err := newFunctionForTest(s.ctx, ast.Log2, primitiveValsToConstants([]interface{}{test.args})...)
		c.Assert(err, IsNil)

		result, err := f.Eval(nil)
		if test.getErr {
			c.Assert(err, NotNil)
		} else {
			c.Assert(err, IsNil)
			if test.isNil {
				c.Assert(result.Kind(), Equals, types.KindNull)
			} else {
				c.Assert(result.GetFloat64(), Equals, test.expect)
			}
		}
	}

	f, err := funcs[ast.Log2].getFunction([]Expression{Zero}, s.ctx)
	c.Assert(err, IsNil)
	c.Assert(f.isDeterministic(), IsTrue)
}

func (s *testEvaluatorSuite) TestLog10(c *C) {
	defer testleak.AfterTest(c)()

	tests := []struct {
		args   interface{}
		expect float64
		isNil  bool
		getErr bool
	}{
		{nil, 0, true, false},
		{int64(100), 2, false, false},
		{float64(100), 2, false, false},
		{int64(101), 2.0043213737826426, false, false},
		{int64(-1), 0, true, false},
		{"100abc", 0, false, true},
	}

	for _, test := range tests {
		f, err := newFunctionForTest(s.ctx, ast.Log10, primitiveValsToConstants([]interface{}{test.args})...)
		c.Assert(err, IsNil)

		result, err := f.Eval(nil)
		if test.getErr {
			c.Assert(err, NotNil)
		} else {
			c.Assert(err, IsNil)
			if test.isNil {
				c.Assert(result.Kind(), Equals, types.KindNull)
			} else {
				c.Assert(result.GetFloat64(), Equals, test.expect)
			}
		}
	}

	f, err := funcs[ast.Log10].getFunction([]Expression{Zero}, s.ctx)
	c.Assert(err, IsNil)
	c.Assert(f.isDeterministic(), IsTrue)
}

func (s *testEvaluatorSuite) TestRand(c *C) {
	defer testleak.AfterTest(c)()
	fc := funcs[ast.Rand]
	f, err := fc.getFunction(nil, s.ctx)
	c.Assert(err, IsNil)
	v, err := f.eval(nil)
	c.Assert(err, IsNil)
	c.Assert(v.GetFloat64(), Less, float64(1))
	c.Assert(v.GetFloat64(), GreaterEqual, float64(0))

	// issue 3211
	f2, err := fc.getFunction([]Expression{&Constant{Value: types.NewIntDatum(20160101), RetType: types.NewFieldType(mysql.TypeLonglong)}}, s.ctx)
	c.Assert(err, IsNil)
	randGen := rand.New(rand.NewSource(20160101))
	for i := 0; i < 3; i++ {
		v, err = f2.eval(nil)
		c.Assert(err, IsNil)
		c.Assert(v.GetFloat64(), Equals, randGen.Float64())
	}
}

func (s *testEvaluatorSuite) TestPow(c *C) {
	defer testleak.AfterTest(c)()
	tbl := []struct {
		Arg []interface{}
		Ret float64
	}{
		{[]interface{}{1, 3}, 1},
		{[]interface{}{2, 2}, 4},
		{[]interface{}{4, 0.5}, 2},
		{[]interface{}{4, -2}, 0.0625},
	}

	Dtbl := tblToDtbl(tbl)

	for _, t := range Dtbl {
		fc := funcs[ast.Pow]
		f, err := fc.getFunction(datumsToConstants(t["Arg"]), s.ctx)
		c.Assert(err, IsNil)
		v, err := f.eval(nil)
		c.Assert(err, IsNil)
		c.Assert(v, testutil.DatumEquals, t["Ret"][0])
	}

	errTbl := []struct {
		Arg []interface{}
	}{
		{[]interface{}{"test", "test"}},
		{[]interface{}{nil, nil}},
		{[]interface{}{1, "test"}},
		{[]interface{}{1, nil}},
		{[]interface{}{10, 700}}, // added overflow test
	}

	errDtbl := tblToDtbl(errTbl)
	for _, t := range errDtbl {
		fc := funcs[ast.Pow]
		f, err := fc.getFunction(datumsToConstants(t["Arg"]), s.ctx)
		c.Assert(err, IsNil)
		_, err = f.eval(nil)
		c.Assert(err, NotNil)
	}
}

func (s *testEvaluatorSuite) TestRound(c *C) {
	defer testleak.AfterTest(c)()
	newDec := types.NewDecFromStringForTest
	tbl := []struct {
		Arg []interface{}
		Ret interface{}
	}{
		{[]interface{}{-1.23}, -1},
		{[]interface{}{-1.23, 0}, -1},
		{[]interface{}{-1.58}, -2},
		{[]interface{}{1.58}, 2},
		{[]interface{}{1.298, 1}, 1.3},
		{[]interface{}{1.298}, 1},
		{[]interface{}{1.298, 0}, 1},
		{[]interface{}{23.298, -1}, 20},
		{[]interface{}{newDec("-1.23")}, newDec("-1")},
		{[]interface{}{newDec("-1.23"), 1}, newDec("-1.2")},
		{[]interface{}{newDec("-1.58")}, newDec("-2")},
		{[]interface{}{newDec("1.58")}, newDec("2")},
		{[]interface{}{newDec("1.58"), 1}, newDec("1.6")},
		{[]interface{}{newDec("23.298"), -1}, newDec("20")},
		{[]interface{}{nil, 2}, nil},
	}

	Dtbl := tblToDtbl(tbl)

	for _, t := range Dtbl {
		fc := funcs[ast.Round]
		f, err := fc.getFunction(datumsToConstants(t["Arg"]), s.ctx)
		c.Assert(err, IsNil)
		v, err := f.eval(nil)
		c.Assert(err, IsNil)
		c.Assert(v, testutil.DatumEquals, t["Ret"][0])
	}
}

func (s *testEvaluatorSuite) TestTruncate(c *C) {
	defer testleak.AfterTest(c)()
	newDec := types.NewDecFromStringForTest
	tbl := []struct {
		Arg []interface{}
		Ret interface{}
	}{
		{[]interface{}{-1.23, 0}, -1},
		{[]interface{}{1.58, 0}, 1},
		{[]interface{}{1.298, 1}, 1.2},
		{[]interface{}{123.2, -1}, 120},
		{[]interface{}{123.2, 100}, 123.2},
		{[]interface{}{123.2, -100}, 0},
		{[]interface{}{123.2, -100}, 0},
		{[]interface{}{1.797693134862315708145274237317043567981e+308, 2},
			1.797693134862315708145274237317043567981e+308},
		{[]interface{}{newDec("-1.23"), 0}, newDec("-1")},
		{[]interface{}{newDec("-1.23"), 1}, newDec("-1.2")},
		{[]interface{}{newDec("-11.23"), -1}, newDec("-10")},
		{[]interface{}{newDec("1.58"), 0}, newDec("1")},
		{[]interface{}{newDec("1.58"), 1}, newDec("1.5")},
		{[]interface{}{newDec("11.58"), -1}, newDec("10")},
		{[]interface{}{newDec("23.298"), -1}, newDec("20")},
		{[]interface{}{newDec("23.298"), -100}, newDec("0")},
		{[]interface{}{newDec("23.298"), 100}, newDec("23.298")},
		{[]interface{}{nil, 2}, nil},
	}

	Dtbl := tblToDtbl(tbl)

	for _, t := range Dtbl {
		fc := funcs[ast.Truncate]
		f, err := fc.getFunction(datumsToConstants(t["Arg"]), s.ctx)
		c.Assert(err, IsNil)
		v, err := f.eval(nil)
		c.Assert(err, IsNil)
		c.Assert(v, testutil.DatumEquals, t["Ret"][0])
	}
}

func (s *testEvaluatorSuite) TestCRC32(c *C) {
	defer testleak.AfterTest(c)()
	tbl := []struct {
		Arg []interface{}
		Ret uint64
	}{
		{[]interface{}{"mysql"}, 2501908538},
		{[]interface{}{"MySQL"}, 3259397556},
		{[]interface{}{"hello"}, 907060870},
	}

	Dtbl := tblToDtbl(tbl)

	for _, t := range Dtbl {
		fc := funcs[ast.CRC32]
		f, err := fc.getFunction(datumsToConstants(t["Arg"]), s.ctx)
		c.Assert(err, IsNil)
		v, err := f.eval(nil)
		c.Assert(err, IsNil)
		c.Assert(v, testutil.DatumEquals, t["Ret"][0])
	}
}

func (s *testEvaluatorSuite) TestConv(c *C) {
	defer testleak.AfterTest(c)()
	tbl := []struct {
		Arg []interface{}
		Ret interface{}
	}{
		{[]interface{}{"a", 16, 2}, "1010"},
		{[]interface{}{"6E", 18, 8}, "172"},
		{[]interface{}{"-17", 10, -18}, "-H"},
		{[]interface{}{"-17", 10, 18}, "2D3FGB0B9CG4BD1H"},
		{[]interface{}{nil, 10, 10}, nil},
		{[]interface{}{"+18aZ", 7, 36}, 1},
		{[]interface{}{"18446744073709551615", -10, 16}, "7FFFFFFFFFFFFFFF"},
		{[]interface{}{"12F", -10, 16}, "C"},
		{[]interface{}{"  FF ", 16, 10}, "255"},
		{[]interface{}{"TIDB", 10, 8}, "0"},
		{[]interface{}{"aa", 10, 2}, "0"},
		{[]interface{}{" A", -10, 16}, "0"},
		{[]interface{}{"a6a", 10, 8}, "0"},
	}

	Dtbl := tblToDtbl(tbl)

	for _, t := range Dtbl {
		fc := funcs[ast.Conv]
		f, err := fc.getFunction(datumsToConstants(t["Arg"]), s.ctx)
		c.Assert(err, IsNil)
		v, err := f.eval(nil)
		c.Assert(err, IsNil)
		c.Assert(v, testutil.DatumEquals, t["Ret"][0])
	}

	v := []struct {
		s    string
		base int64
		ret  string
	}{
		{"-123456D1f", 5, "-1234"},
		{"+12azD", 16, "12a"},
		{"+", 12, ""},
	}
	for _, t := range v {
		r := getValidPrefix(t.s, t.base)
		c.Assert(r, Equals, t.ret)
	}
}

func (s *testEvaluatorSuite) TestSign(c *C) {
	defer testleak.AfterTest(c)()

	for _, t := range []struct {
		num interface{}
		ret interface{}
		err Checker
	}{
		{nil, nil, IsNil},
		{1, 1, IsNil},
		{0, 0, IsNil},
		{-1, -1, IsNil},
		{0.4, 1, IsNil},
		{-0.4, -1, IsNil},
		{"1", 1, IsNil},
		{"-1", -1, IsNil},
		{"1a", 1, NotNil},
		{"-1a", -1, NotNil},
		{"a", 0, NotNil},
		{uint64(9223372036854775808), 1, IsNil},
	} {
		fc := funcs[ast.Sign]
		f, err := fc.getFunction(datumsToConstants(types.MakeDatums(t.num)), s.ctx)
		c.Assert(err, IsNil)
		v, err := f.eval(nil)
		c.Assert(err, t.err)
		c.Assert(v, testutil.DatumEquals, types.NewDatum(t.ret))
	}
}

func (s *testEvaluatorSuite) TestDegrees(c *C) {
	defer testleak.AfterTest(c)()
	sc := s.ctx.GetSessionVars().StmtCtx
	sc.IgnoreTruncate = false
	cases := []struct {
		args     interface{}
		expected float64
		isNil    bool
		getErr   bool
	}{
		{nil, 0, true, false},
		{int64(0), float64(0), false, false},
		{int64(1), float64(57.29577951308232), false, false},
		{float64(1), float64(57.29577951308232), false, false},
		{float64(math.Pi), float64(180), false, false},
		{float64(-math.Pi / 2), float64(-90), false, false},
		{"", float64(0), false, true},
		{"-2", float64(-114.59155902616465), false, false},
		{"abc", float64(0), false, true},
		{"+1abc", 57.29577951308232, false, true},
	}

	for _, t := range cases {
		f, err := newFunctionForTest(s.ctx, ast.Degrees, primitiveValsToConstants([]interface{}{t.args})...)
		c.Assert(err, IsNil)
		d, err := f.Eval(nil)
		if t.getErr {
			c.Assert(err, NotNil)
		} else {
			c.Assert(err, IsNil)
			if t.isNil {
				c.Assert(d.Kind(), Equals, types.KindNull)
			} else {
				c.Assert(d.GetFloat64(), Equals, t.expected)
			}
		}
	}
	f, err := funcs[ast.Degrees].getFunction([]Expression{Zero}, s.ctx)
	c.Assert(err, IsNil)
	c.Assert(f.isDeterministic(), IsTrue)
}

func (s *testEvaluatorSuite) TestSqrt(c *C) {
	defer testleak.AfterTest(c)()
	tbl := []struct {
		Arg interface{}
		Ret interface{}
	}{
		{nil, nil},
		{int64(1), float64(1)},
		{float64(4), float64(2)},
		{"4", float64(2)},
		{"9", float64(3)},
		{"-16", nil},
	}

	Dtbl := tblToDtbl(tbl)

	for _, t := range Dtbl {
		fc := funcs[ast.Sqrt]
		f, err := fc.getFunction(datumsToConstants(t["Arg"]), s.ctx)
		c.Assert(err, IsNil)
		v, err := f.eval(nil)
		c.Assert(err, IsNil)
		c.Assert(v, DeepEquals, t["Ret"][0], Commentf("arg:%v", t["Arg"]))
	}
}

func (s *testEvaluatorSuite) TestPi(c *C) {
	defer testleak.AfterTest(c)()
	fc := funcs[ast.PI]
	f, _ := fc.getFunction(nil, s.ctx)
	pi, err := f.eval(nil)
	c.Assert(err, IsNil)
	c.Assert(pi, testutil.DatumEquals, types.NewDatum(math.Pi))
}

func (s *testEvaluatorSuite) TestRadians(c *C) {
	defer testleak.AfterTest(c)()
	tbl := []struct {
		Arg interface{}
		Ret interface{}
	}{
		{nil, nil},
		{0, float64(0)},
		{float64(180), float64(math.Pi)},
		{-360, -2 * float64(math.Pi)},
		{"180", float64(math.Pi)},
	}

	Dtbl := tblToDtbl(tbl)
	for _, t := range Dtbl {
		fc := funcs[ast.Radians]
		f, err := fc.getFunction(datumsToConstants(t["Arg"]), s.ctx)
		c.Assert(err, IsNil)
		v, err := f.eval(nil)
		c.Assert(err, IsNil)
		c.Assert(v, testutil.DatumEquals, t["Ret"][0])
	}

	invalidArg := "notNum"
	fc := funcs[ast.Radians]
	f, err := fc.getFunction(datumsToConstants([]types.Datum{types.NewDatum(invalidArg)}), s.ctx)
	c.Assert(err, IsNil)
	_, err = f.eval(nil)
	c.Assert(err, NotNil)
}

func (s *testEvaluatorSuite) TestSin(c *C) {
	defer testleak.AfterTest(c)()
	cases := []struct {
		args     interface{}
		expected float64
		isNil    bool
		getErr   bool
	}{
		{nil, 0, true, false},
		{int64(0), float64(0), false, false},
		{math.Pi, float64(math.Sin(math.Pi)), false, false}, // Pie ==> 0
		{-math.Pi, float64(math.Sin(-math.Pi)), false, false},
		{math.Pi / 2, float64(math.Sin(math.Pi / 2)), false, false}, // Pie/2 ==> 1
		{-math.Pi / 2, float64(math.Sin(-math.Pi / 2)), false, false},
		{math.Pi / 6, float64(math.Sin(math.Pi / 6)), false, false}, // Pie/6(30 degrees) ==> 0.5
		{-math.Pi / 6, float64(math.Sin(-math.Pi / 6)), false, false},
		{math.Pi * 2, float64(math.Sin(math.Pi * 2)), false, false},
		{string("adfsdfgs"), 0, false, true},
		{"0.000", 0, false, false},
	}

	for _, t := range cases {
		f, err := newFunctionForTest(s.ctx, ast.Sin, primitiveValsToConstants([]interface{}{t.args})...)
		c.Assert(err, IsNil)

		d, err := f.Eval(nil)
		if t.getErr {
			c.Assert(err, NotNil)
		} else {
			c.Assert(err, IsNil)
			if t.isNil {
				c.Assert(d.Kind(), Equals, types.KindNull)
			} else {
				c.Assert(d.GetFloat64(), Equals, t.expected)
			}
		}
	}

	f, err := funcs[ast.Sin].getFunction([]Expression{Zero}, s.ctx)
	c.Assert(err, IsNil)
	c.Assert(f.isDeterministic(), IsTrue)
}

func (s *testEvaluatorSuite) TestCos(c *C) {
	defer testleak.AfterTest(c)()
	cases := []struct {
		args     interface{}
		expected float64
		isNil    bool
		getErr   bool
	}{
		{nil, 0, true, false},
		{int64(0), float64(1), false, false},
		{math.Pi, float64(-1), false, false}, // cos pi equals -1
		{-math.Pi, float64(-1), false, false},
		{math.Pi / 2, float64(math.Cos(math.Pi / 2)), false, false}, // Pi/2 is some near 0 (6.123233995736766e-17) but not 0. Even in math it is 0.
		{-math.Pi / 2, float64(math.Cos(-math.Pi / 2)), false, false},
		{"0.000", float64(1), false, false}, // string value case
		{"sdfgsfsdf", float64(0), false, true},
	}

	for _, t := range cases {
		f, err := newFunctionForTest(s.ctx, ast.Cos, primitiveValsToConstants([]interface{}{t.args})...)
		c.Assert(err, IsNil)

		d, err := f.Eval(nil)
		if t.getErr {
			c.Assert(err, NotNil)
		} else {
			c.Assert(err, IsNil)
			if t.isNil {
				c.Assert(d.Kind(), Equals, types.KindNull)
			} else {
				c.Assert(d.GetFloat64(), Equals, t.expected)
			}
		}
	}

	f, err := funcs[ast.Cos].getFunction([]Expression{Zero}, s.ctx)
	c.Assert(err, IsNil)
	c.Assert(f.isDeterministic(), IsTrue)
}

func (s *testEvaluatorSuite) TestAcos(c *C) {
	defer testleak.AfterTest(c)()

	tests := []struct {
		args   interface{}
		expect float64
		isNil  bool
		getErr bool
	}{
		{nil, 0, true, false},
		{float64(1), 0, false, false},
		{float64(2), 0, true, false},
		{float64(-1), 3.141592653589793, false, false},
		{float64(-2), 0, true, false},
		{"tidb", 0, false, true},
	}

	for _, test := range tests {
		f, err := newFunctionForTest(s.ctx, ast.Acos, primitiveValsToConstants([]interface{}{test.args})...)
		c.Assert(err, IsNil)

		result, err := f.Eval(nil)
		if test.getErr {
			c.Assert(err, NotNil)
		} else {
			c.Assert(err, IsNil)
			if test.isNil {
				c.Assert(result.Kind(), Equals, types.KindNull)
			} else {
				c.Assert(result.GetFloat64(), Equals, test.expect)
			}
		}
	}

	f, err := funcs[ast.Acos].getFunction([]Expression{Zero}, s.ctx)
	c.Assert(err, IsNil)
	c.Assert(f.isDeterministic(), IsTrue)
}

func (s *testEvaluatorSuite) TestAsin(c *C) {
	defer testleak.AfterTest(c)()

	tests := []struct {
		args   interface{}
		expect float64
		isNil  bool
		getErr bool
	}{
		{nil, 0, true, false},
		{float64(1), 1.5707963267948966, false, false},
		{float64(2), 0, true, false},
		{float64(-1), -1.5707963267948966, false, false},
		{float64(-2), 0, true, false},
		{"tidb", 0, false, true},
	}

	for _, test := range tests {
		f, err := newFunctionForTest(s.ctx, ast.Asin, primitiveValsToConstants([]interface{}{test.args})...)
		c.Assert(err, IsNil)

		result, err := f.Eval(nil)
		if test.getErr {
			c.Assert(err, NotNil)
		} else {
			c.Assert(err, IsNil)
			if test.isNil {
				c.Assert(result.Kind(), Equals, types.KindNull)
			} else {
				c.Assert(result.GetFloat64(), Equals, test.expect)
			}
		}
	}

	f, err := funcs[ast.Asin].getFunction([]Expression{Zero}, s.ctx)
	c.Assert(err, IsNil)
	c.Assert(f.isDeterministic(), IsTrue)
}

func (s *testEvaluatorSuite) TestAtan(c *C) {
	defer testleak.AfterTest(c)()

	tests := []struct {
		args   []interface{}
		expect float64
		isNil  bool
		getErr bool
	}{
		{[]interface{}{nil}, 0, true, false},
		{[]interface{}{nil, nil}, 0, true, false},
		{[]interface{}{float64(1)}, 0.7853981633974483, false, false},
		{[]interface{}{float64(-1)}, -0.7853981633974483, false, false},
		{[]interface{}{float64(0), float64(-2)}, float64(math.Pi), false, false},
		{[]interface{}{"tidb"}, 0, false, true},
	}

	for _, test := range tests {
		f, err := newFunctionForTest(s.ctx, ast.Atan, primitiveValsToConstants(test.args)...)
		c.Assert(err, IsNil)

		result, err := f.Eval(nil)
		if test.getErr {
			c.Assert(err, NotNil)
		} else {
			c.Assert(err, IsNil)
			if test.isNil {
				c.Assert(result.Kind(), Equals, types.KindNull)
			} else {
				c.Assert(result.GetFloat64(), Equals, test.expect)
			}
		}
	}

	f, err := funcs[ast.Atan].getFunction([]Expression{Zero}, s.ctx)
	c.Assert(err, IsNil)
	c.Assert(f.isDeterministic(), IsTrue)
}

func (s *testEvaluatorSuite) TestTan(c *C) {
	defer testleak.AfterTest(c)()
	cases := []struct {
		args     interface{}
		expected float64
		isNil    bool
		getErr   bool
	}{
		{nil, 0, true, false},
		{int64(0), float64(0), false, false},
		{math.Pi / 4, float64(1), false, false},
		{-math.Pi / 4, float64(-1), false, false},
		{math.Pi * 3 / 4, math.Tan(math.Pi * 3 / 4), false, false}, //in mysql and golang, it equals -1.0000000000000002, not -1
		{"0.000", float64(0), false, false},
		{"sdfgsdfg", 0, false, true},
	}

	for _, t := range cases {
		f, err := newFunctionForTest(s.ctx, ast.Tan, primitiveValsToConstants([]interface{}{t.args})...)
		c.Assert(err, IsNil)

		d, err := f.Eval(nil)
		if t.getErr {
			c.Assert(err, NotNil)
		} else {
			c.Assert(err, IsNil)
			if t.isNil {
				c.Assert(d.Kind(), Equals, types.KindNull)
			} else {
				c.Assert(d.GetFloat64(), Equals, t.expected)
			}
		}
	}

	f, err := funcs[ast.Tan].getFunction([]Expression{Zero}, s.ctx)
	c.Assert(err, IsNil)
	c.Assert(f.isDeterministic(), IsTrue)
}

func (s *testEvaluatorSuite) TestCot(c *C) {
	defer testleak.AfterTest(c)()
	tbl := []struct {
		Arg interface{}
		Ret interface{}
	}{
		{nil, nil},
		{math.Pi / 4, math.Cos(math.Pi/4) / math.Sin(math.Pi/4)}, // cot pi/4 does not return 1 actually
		{-math.Pi / 4, math.Cos(-math.Pi/4) / math.Sin(-math.Pi/4)},
		{math.Pi * 3 / 4, math.Cos(math.Pi*3/4) / math.Sin(math.Pi*3/4)},
		{"3.1415926", math.Cos(3.1415926) / math.Sin(3.1415926)},
	}

	Dtbl := tblToDtbl(tbl)
	for _, t := range Dtbl {
		fc := funcs[ast.Cot]
		f, err := fc.getFunction(datumsToConstants(t["Arg"]), s.ctx)
		c.Assert(err, IsNil)
		v, err := f.eval(nil)
		c.Assert(err, IsNil)
		c.Log(t)
		c.Assert(v, testutil.DatumEquals, t["Ret"][0])
	}
}
