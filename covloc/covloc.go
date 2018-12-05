package covloc

import (
	"errors"
	"sort"

	"github.com/coordination-institute/debugging-tools/ast"
	"github.com/coordination-institute/debugging-tools/srclocation"
)

type CoverageLoc struct {
	HitCount      int
	CoverageRange srclocation.SourceLocation
	SrcLocs       []srclocation.SourceLocation
}

func Get(filename string) ([]CoverageLoc, error) {
	plainAST, err := ast.Get(filename)
	if err != nil {
		return []CoverageLoc{}, err
	}

	covLocs, err := ToCoverageLocs(plainAST)
	if err != nil {
		return []CoverageLoc{}, err
	}
	return covLocs, nil
	// sort coverageLocs by .CoverageRange.ByteOffset
}

func ToCoverageLocs(node ast.ASTTree) ([]CoverageLoc, error) {
	var covLocs []CoverageLoc
	var covLoc CoverageLoc
	covLoc.CoverageRange = node.SrcLoc

	covLoc.SrcLocs = append(covLoc.SrcLocs, node.SrcLoc)
	children := node.Children
	sort.Slice(children, func(i, j int) bool {
		// If they have the same byte offset, my current belief is that that can
		// only happen if one of them is empty. In that case, we're going to throw
		// it away anyway.
		return children[i].SrcLoc.ByteOffset < children[j].SrcLoc.ByteOffset
	})
	for _, childNode := range children {
		if childNode.SrcLoc.ByteLength == 0 {
			// This node has no representation in the source file and neither could its children.
			continue
		} else if node.SrcLoc.ByteLength == childNode.SrcLoc.ByteLength {
			// This node is equal in size to its parent. Any siblings must be empty.
			// Not sure what to do here yet, let's roll with it and see what happens.
		}

		rightSrcLoc := covLoc.SrcLocs[len(covLoc.SrcLocs)-1]

		byteOffset1 := rightSrcLoc.ByteOffset
		byteLength1 := childNode.SrcLoc.ByteOffset - rightSrcLoc.ByteOffset
		if byteLength1 < 0 {
			return covLocs, errors.New("Negative byte length1")
		}
		byteOffset2 := childNode.SrcLoc.ByteOffset + childNode.SrcLoc.ByteLength
		byteLength2 := rightSrcLoc.ByteOffset + rightSrcLoc.ByteLength - byteOffset2
		if byteLength2 < 0 {
			return covLocs, errors.New("Negative byte length2")
		}

		// Cut off the last one, since we're splitting it in two.
		covLoc.SrcLocs = covLoc.SrcLocs[:len(covLoc.SrcLocs)-1]

		if byteLength1 != 0 {
			covLoc.SrcLocs = append(covLoc.SrcLocs, srclocation.SourceLocation{
				byteOffset1,
				byteLength1,
				node.SrcLoc.SourceFileName,
				*new(rune),
			})
		}

		if byteLength2 != 0 {
			covLoc.SrcLocs = append(covLoc.SrcLocs, srclocation.SourceLocation{
				byteOffset2,
				byteLength2,
				node.SrcLoc.SourceFileName,
				*new(rune),
			})
		}

		childCovLocs, err := ToCoverageLocs(*childNode)
		if err == nil {
			covLocs = append(covLocs, childCovLocs...)
		}

	}

	covLocs = append(covLocs, covLoc)
	sort.Slice(covLocs, func(i, j int) bool {
		if covLocs[i].CoverageRange.ByteOffset == covLocs[j].CoverageRange.ByteOffset {
			return covLocs[i].CoverageRange.ByteLength < covLocs[j].CoverageRange.ByteLength
		}
		return covLocs[i].CoverageRange.ByteOffset < covLocs[j].CoverageRange.ByteOffset
	})
	return covLocs, nil
}

type CoverageCount struct {
	SrcLoc srclocation.SourceLocation
	Count  int
}
