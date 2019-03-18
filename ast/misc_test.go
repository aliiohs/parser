// Copyright 2016 PingCAP, Inc.
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

package ast_test

import (
	. "github.com/pingcap/check"
	"github.com/pingcap/parser"
	. "github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/auth"
)

var _ = Suite(&testMiscSuite{})

type testMiscSuite struct {
}

type visitor struct{}

func (v visitor) Enter(in Node) (Node, bool) {
	return in, false
}

func (v visitor) Leave(in Node) (Node, bool) {
	return in, true
}

type visitor1 struct {
	visitor
}

func (visitor1) Enter(in Node) (Node, bool) {
	return in, true
}

func (ts *testMiscSuite) TestMiscVisitorCover(c *C) {
	valueExpr := NewValueExpr(42)
	stmts := []Node{
		&AdminStmt{},
		&AlterUserStmt{},
		&BeginStmt{},
		&BinlogStmt{},
		&CommitStmt{},
		&CreateUserStmt{},
		&DeallocateStmt{},
		&DoStmt{},
		&ExecuteStmt{UsingVars: []ExprNode{valueExpr}},
		&ExplainStmt{Stmt: &ShowStmt{}},
		&GrantStmt{},
		&PrepareStmt{SQLVar: &VariableExpr{Value: valueExpr}},
		&RollbackStmt{},
		&SetPwdStmt{},
		&SetStmt{Variables: []*VariableAssignment{
			{
				Value: valueExpr,
			},
		}},
		&UseStmt{},
		&AnalyzeTableStmt{
			TableNames: []*TableName{
				{},
			},
		},
		&FlushStmt{},
		&PrivElem{},
		&VariableAssignment{Value: valueExpr},
		&KillStmt{},
		&DropStatsStmt{Table: &TableName{}},
	}

	for _, v := range stmts {
		v.Accept(visitor{})
		v.Accept(visitor1{})
	}
}

func (ts *testMiscSuite) TestDDLVisitorCover(c *C) {
	sql := `
create table t (c1 smallint unsigned, c2 int unsigned);
alter table t add column a smallint unsigned after b;
create index t_i on t (id);
create database test character set utf8;
drop database test;
drop index t_i on t;
drop table t;
truncate t;
create table t (
jobAbbr char(4) not null,
constraint foreign key (jobabbr) references ffxi_jobtype (jobabbr) on delete cascade on update cascade
);
`
	parse := parser.New()
	stmts, _, err := parse.Parse(sql, "", "")
	c.Assert(err, IsNil)
	for _, stmt := range stmts {
		stmt.Accept(visitor{})
		stmt.Accept(visitor1{})
	}
}

func (ts *testMiscSuite) TestDMLVistorCover(c *C) {
	sql := `delete from somelog where user = 'jcole' order by timestamp_column limit 1;
delete t1, t2 from t1 inner join t2 inner join t3 where t1.id=t2.id and t2.id=t3.id;
select * from t where exists(select * from t k where t.c = k.c having sum(c) = 1);
insert into t_copy select * from t where t.x > 5;
(select /*+ TIDB_INLJ(t1) */ a from t1 where a=10 and b=1) union (select /*+ TIDB_SMJ(t2) */ a from t2 where a=11 and b=2) order by a limit 10;
update t1 set col1 = col1 + 1, col2 = col1;
show create table t;
load data infile '/tmp/t.csv' into table t fields terminated by 'ab' enclosed by 'b';`

	p := parser.New()
	stmts, _, err := p.Parse(sql, "", "")
	c.Assert(err, IsNil)
	for _, stmt := range stmts {
		stmt.Accept(visitor{})
		stmt.Accept(visitor1{})
	}
}
func (ts *testMiscSuite) TestChangeStmt(c *C) {
	sql := `change pump to pump_state='paused' for NodeID "127.0.0.1:8249";`

	p := parser.New()
	stmts, _, err := p.Parse(sql, "", "")
	c.Assert(err, IsNil)
	for _, stmt := range stmts {
		stmt.Accept(visitor{})
		stmt.Accept(visitor1{})
	}
}

func (ts *testMiscSuite) TestSensitiveStatement(c *C) {
	positive := []StmtNode{
		&SetPwdStmt{},
		&CreateUserStmt{},
		&AlterUserStmt{},
		&GrantStmt{},
	}
	for i, stmt := range positive {
		_, ok := stmt.(SensitiveStmtNode)
		c.Assert(ok, IsTrue, Commentf("%d, %#v fail", i, stmt))
	}

	negative := []StmtNode{
		&DropUserStmt{},
		&RevokeStmt{},
		&AlterTableStmt{},
		&CreateDatabaseStmt{},
		&CreateIndexStmt{},
		&CreateTableStmt{},
		&DropDatabaseStmt{},
		&DropIndexStmt{},
		&DropTableStmt{},
		&RenameTableStmt{},
		&TruncateTableStmt{},
	}
	for _, stmt := range negative {
		_, ok := stmt.(SensitiveStmtNode)
		c.Assert(ok, IsFalse)
	}
}

func (ts *testMiscSuite) TestUserSpec(c *C) {
	hashString := "*3D56A309CD04FA2EEF181462E59011F075C89548"
	u := UserSpec{
		User: &auth.UserIdentity{
			Username: "test",
		},
		AuthOpt: &AuthOption{
			ByAuthString: false,
			AuthString:   "xxx",
			HashString:   hashString,
		},
	}
	pwd, ok := u.EncodedPassword()
	c.Assert(ok, IsTrue)
	c.Assert(pwd, Equals, u.AuthOpt.HashString)

	u.AuthOpt.HashString = "not-good-password-format"
	pwd, ok = u.EncodedPassword()
	c.Assert(ok, IsFalse)

	u.AuthOpt.ByAuthString = true
	pwd, ok = u.EncodedPassword()
	c.Assert(ok, IsTrue)
	c.Assert(pwd, Equals, hashString)

	u.AuthOpt.AuthString = ""
	pwd, ok = u.EncodedPassword()
	c.Assert(ok, IsTrue)
	c.Assert(pwd, Equals, "")
}

func (ts *testMiscSuite) TestTableOptimizerHintRestore(c *C) {
	testCases := []NodeRestoreTestCase{
		{"TIDB_SMJ(`t1`)", "TIDB_SMJ(`t1`)"},
		{"TIDB_SMJ(t1)", "TIDB_SMJ(`t1`)"},
		{"TIDB_SMJ(t1,t2)", "TIDB_SMJ(`t1`, `t2`)"},
		{"TIDB_INLJ(t1,t2)", "TIDB_INLJ(`t1`, `t2`)"},
		{"TIDB_HJ(t1,t2)", "TIDB_HJ(`t1`, `t2`)"},
		{"MAX_EXECUTION_TIME(3000)", "MAX_EXECUTION_TIME(3000)"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).TableHints[0]
	}
	RunNodeRestoreTest(c, testCases, "select /*+ %s */ * from t1 join t2", extractNodeFunc)
}
