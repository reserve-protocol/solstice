package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/reserve-protocol/solstice/ast"
	"github.com/reserve-protocol/solstice/common"
	"github.com/reserve-protocol/solstice/covloc"
	"github.com/reserve-protocol/solstice/srclocation"
)

func AssertCovLocsEqual(t *testing.T, covlocs1 []covloc.CoverageLoc, covlocs2 []covloc.CoverageLoc) {
	if len(covlocs1) != len(covlocs2) {
		t.Errorf("Length was %d instead of %d", len(covlocs1), len(covlocs2))
		return
	}

	for index, element := range covlocs1 {
		if element.HitCount != covlocs2[index].HitCount {
			t.Errorf("CovLocs HitCount differed at index %d\nFirst element was  %v\nSecond element was %v", index, element, covlocs2[index])
		}

		if element.CoverageRange != covlocs2[index].CoverageRange {
			t.Errorf("CovLocs CoverageRange differed at index %d\nFirst element was  %v\nSecond element was %v", index, element, covlocs2[index])
		}

		AssertSrcLocsEqual(t, element.SrcLocs, covlocs2[index].SrcLocs)
	}
}

func AssertSrcLocsEqual(t *testing.T, srclocs1 []srclocation.SourceLocation, srclocs2 []srclocation.SourceLocation) {
	if len(srclocs1) != len(srclocs2) {
		t.Errorf("Length was %d instead of %d", len(srclocs1), len(srclocs2))
		return
	}

	for index, element := range srclocs1 {
		if !cmp.Equal(element, srclocs2[index]) {
			t.Errorf("SrcLocs differed at index %d\nFirst element was  %v\nSecond element was %v", index, element, srclocs2[index])
		}
	}
}

func TestNodeNoChildren(t *testing.T) {
	onlyNode := srclocation.SourceLocation{ByteOffset: 0, ByteLength: 3}

	testTree := ast.AST{
		SrcLoc: onlyNode,
		Children: []*ast.AST{},
	}

	gotCovlocs, err := covloc.ToCoverageLocs(testTree)
	common.Check(err)
	wantCovlocs := []covloc.CoverageLoc{
		{
			CoverageRange: onlyNode,
			SrcLocs:       []srclocation.SourceLocation{onlyNode},
		},
	}

	AssertCovLocsEqual(t, gotCovlocs, wantCovlocs)
}

func TestNodeChildSameSize(t *testing.T) {
	node := srclocation.SourceLocation{ByteOffset: 0, ByteLength: 3}

	testTree := ast.AST{
		SrcLoc:   node,
		Children: []*ast.AST{{SrcLoc: node}},
	}

	gotCovlocs, err := covloc.ToCoverageLocs(testTree)
	common.Check(err)
	wantCovlocs := []covloc.CoverageLoc{
		{
			CoverageRange: node,
			SrcLocs:       []srclocation.SourceLocation{node},
		},
	}

	AssertCovLocsEqual(t, gotCovlocs, wantCovlocs)
}

func TestNodeChildrenPartition(t *testing.T) {
	topNode   := srclocation.SourceLocation{ByteOffset: 0, ByteLength: 2}
	leftNode  := srclocation.SourceLocation{ByteOffset: 0, ByteLength: 1}
	rightNode := srclocation.SourceLocation{ByteOffset: 1, ByteLength: 1}

	testTree := ast.AST{
		SrcLoc:   topNode,
		Children: []*ast.AST{
			{SrcLoc: leftNode},
			{SrcLoc: rightNode},
		},
	}

	gotCovlocs, err := covloc.ToCoverageLocs(testTree)
	common.Check(err)
	wantCovlocs := []covloc.CoverageLoc{
		{
			CoverageRange: leftNode,
			SrcLocs:       []srclocation.SourceLocation{leftNode},
		},
		{
			CoverageRange: rightNode,
			SrcLocs:       []srclocation.SourceLocation{rightNode},
		},
	}

	AssertCovLocsEqual(t, gotCovlocs, wantCovlocs)
}

func TestNodeSplitTwoBranches(t *testing.T) {
	topNode := srclocation.SourceLocation{ByteOffset: 0, ByteLength: 3}
	leftNode := srclocation.SourceLocation{ByteOffset: 0, ByteLength: 1}
	rightNode := srclocation.SourceLocation{ByteOffset: 2, ByteLength: 1}

	testTree := ast.AST{
		SrcLoc: topNode,
		Children: []*ast.AST{
			{SrcLoc: leftNode},
			{SrcLoc: rightNode},
		},
	}

	gotCovlocs, err := covloc.ToCoverageLocs(testTree)
	common.Check(err)
	wantCovlocs := []covloc.CoverageLoc{
		{
			CoverageRange: leftNode,
			SrcLocs:       []srclocation.SourceLocation{leftNode},
		},
		{
			CoverageRange: topNode,
			SrcLocs:       []srclocation.SourceLocation{{ByteOffset: 1, ByteLength: 1}},
		},
		{
			CoverageRange: rightNode,
			SrcLocs:       []srclocation.SourceLocation{rightNode},
		},
	}

	AssertCovLocsEqual(t, gotCovlocs, wantCovlocs)
}

func TestNodeSplitMiddleChild(t *testing.T) {
	topNode := srclocation.SourceLocation{ByteOffset: 0, ByteLength: 3}
	middleNode := srclocation.SourceLocation{ByteOffset: 1, ByteLength: 1}

	testTree := ast.AST{
		SrcLoc:   topNode,
		Children: []*ast.AST{{SrcLoc: middleNode}},
	}

	gotCovlocs, err := covloc.ToCoverageLocs(testTree)
	common.Check(err)
	wantCovlocs := []covloc.CoverageLoc{
		{
			CoverageRange: topNode,
			SrcLocs:       []srclocation.SourceLocation{
				{ByteOffset: 0, ByteLength: 1},
				{ByteOffset: 2, ByteLength: 1},
			},
		},
		{
			CoverageRange: middleNode,
			SrcLocs:       []srclocation.SourceLocation{{ByteOffset: 1, ByteLength: 1}},
		},
	}

	AssertCovLocsEqual(t, gotCovlocs, wantCovlocs)
}

func TestNodeSplitLeftSide(t *testing.T) {
	topNode := srclocation.SourceLocation{ByteOffset: 0, ByteLength: 2}
	sideNode := srclocation.SourceLocation{ByteOffset: 0, ByteLength: 1}

	testTree := ast.AST{
		SrcLoc:   topNode,
		Children: []*ast.AST{{SrcLoc: sideNode}},
	}

	gotCovlocs, err := covloc.ToCoverageLocs(testTree)
	common.Check(err)
	wantCovlocs := []covloc.CoverageLoc{
		{
			CoverageRange: sideNode,
			SrcLocs:       []srclocation.SourceLocation{sideNode},
		},
		{
			CoverageRange: topNode,
			SrcLocs:       []srclocation.SourceLocation{{ByteOffset: 1, ByteLength: 1}},
		},
	}

	AssertCovLocsEqual(t, gotCovlocs, wantCovlocs)
}

func TestNodeSplitRightSide(t *testing.T) {
	topNode := srclocation.SourceLocation{ByteOffset: 0, ByteLength: 2}
	sideNode := srclocation.SourceLocation{ByteOffset: 1, ByteLength: 1}

	testTree := ast.AST{
		SrcLoc:   topNode,
		Children: []*ast.AST{{SrcLoc: sideNode}},
	}

	gotCovlocs, err := covloc.ToCoverageLocs(testTree)
	common.Check(err)
	wantCovlocs := []covloc.CoverageLoc{
		{
			CoverageRange: topNode,
			SrcLocs:       []srclocation.SourceLocation{{ByteOffset: 0, ByteLength: 1}},
		},
		{
			CoverageRange: sideNode,
			SrcLocs:       []srclocation.SourceLocation{sideNode},
		},
	}

	AssertCovLocsEqual(t, gotCovlocs, wantCovlocs)
}
