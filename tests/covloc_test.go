package main

import (
    "fmt"
    "testing"

    "github.com/google/go-cmp/cmp"

    "github.com/coordination-institute/debugging-tools/ast"
    "github.com/coordination-institute/debugging-tools/common"
    "github.com/coordination-institute/debugging-tools/covloc"
    "github.com/coordination-institute/debugging-tools/srclocation"
)

func TestNodeSplitting(t *testing.T) {
    topNode   := srclocation.SourceLocation{ByteOffset: 0, ByteLength: 3}
    leftNode  := srclocation.SourceLocation{ByteOffset: 0, ByteLength: 1}
    rightNode := srclocation.SourceLocation{ByteOffset: 2, ByteLength: 1}

    testTree := ast.ASTTree{
        SrcLoc:   topNode,
        Children: []*ast.ASTTree{
            {SrcLoc: leftNode},
            {SrcLoc: rightNode},
        },
    }

    gotCovlocs, err := covloc.ToCoverageLocs(testTree)
    common.Check(err)
    wantCovlocs := []covloc.CoverageLoc{
        {
            HitCount:      0,
            CoverageRange: leftNode,
            SrcLocs:       []srclocation.SourceLocation{leftNode},
        },
        {
            HitCount:      0,
            CoverageRange: topNode,
            SrcLocs:       []srclocation.SourceLocation{{ByteOffset: 1, ByteLength: 1}},
        },
        {
            HitCount:      0,
            CoverageRange: rightNode,
            SrcLocs:       []srclocation.SourceLocation{rightNode},
        },
    }

    AssertCovLocsEqual(t, gotCovlocs, wantCovlocs)
}

func TestNodeSplitting2(t *testing.T) {
    topNode    := srclocation.SourceLocation{ByteOffset: 0, ByteLength: 3}
    middleNode := srclocation.SourceLocation{ByteOffset: 1, ByteLength: 1}

    testTree := ast.ASTTree{
        SrcLoc:   topNode,
        Children: []*ast.ASTTree{{SrcLoc: middleNode}},
    }

    gotCovlocs, err := covloc.ToCoverageLocs(testTree)
    common.Check(err)
    wantCovlocs := []covloc.CoverageLoc{
        {
            HitCount:      0,
            CoverageRange: topNode,
            SrcLocs:       []srclocation.SourceLocation{
                {ByteOffset: 0, ByteLength: 1},
                {ByteOffset: 2, ByteLength: 1},
            },
        },
        {
            HitCount:      0,
            CoverageRange: middleNode,
            SrcLocs:       []srclocation.SourceLocation{{ByteOffset: 1, ByteLength: 1}},
        },
    }

    AssertCovLocsEqual(t, gotCovlocs, wantCovlocs)
}

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
       fmt.Println(srclocs1)
       fmt.Println("---------------")
       fmt.Println(srclocs2)
       return
    }

    for index, element := range srclocs1 {
        if !cmp.Equal(element, srclocs2[index]) {
           t.Errorf("SrcLocs differed at index %d\nFirst element was  %v\nSecond element was %v", index, element, srclocs2[index])
        }
    }
}
