package covloc

import (
	"errors"
	"sort"

	"github.com/reserve-protocol/solstice/ast"
	"github.com/reserve-protocol/solstice/srclocation"
)

// Coverage hits from execution traces will be associated with a single srcloc
// byte range. But that hit only represents execution of the lowest-precedence
// operation in that byte range; other operations will be represented by
// smaller byte ranges within it. In order not to double-count those, we break
// apart the srcloc into the ranges that are uniquely represented by it. Those
// are stored in SrcLocs, and the original is stored in CoverageRange.
// HitCount accumulates hits from the execution traces.
type CoverageLoc struct {
	HitCount      int
	CoverageRange srclocation.SourceLocation
	SrcLocs       []srclocation.SourceLocation
}

func ToCoverageLocs(node ast.AST) ([]CoverageLoc, error) {
	var covLocs []CoverageLoc // The cumulative covlocs of the whole tree.
	var covLoc CoverageLoc // The covloc represented by the top node of the input tree.
	covLoc.CoverageRange = node.SrcLoc
	covLoc.SrcLocs = append(covLoc.SrcLocs, node.SrcLoc)

	sort.Slice(node.Children, func(i, j int) bool {
		// If they have the same byte offset, my current belief is that that can
		// only happen if one of them is empty. In that case, we're going to throw
		// it away anyway.
		return node.Children[i].SrcLoc.ByteOffset < node.Children[j].SrcLoc.ByteOffset
	})

	for _, childNode := range node.Children {
		if childNode.SrcLoc.ByteLength == 0 {
			// This node has no representation in the source file and neither could its children.
			continue
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

		// Cut off the last srcloc, since we're splitting it in two.
		covLoc.SrcLocs = covLoc.SrcLocs[:len(covLoc.SrcLocs)-1]

		if byteLength1 != 0 {
			covLoc.SrcLocs = append(covLoc.SrcLocs, srclocation.SourceLocation{
				byteOffset1,
				byteLength1,
				node.SrcLoc.SourceFileName,
				rune(0),
			})
		}

		if byteLength2 != 0 {
			covLoc.SrcLocs = append(covLoc.SrcLocs, srclocation.SourceLocation{
				byteOffset2,
				byteLength2,
				node.SrcLoc.SourceFileName,
				rune(0),
			})
		}

		childCovLocs, err := ToCoverageLocs(*childNode)
		if err == nil {
			covLocs = append(covLocs, childCovLocs...)
		}
	}

	// If the children completely partitian the parent, we don't want to include the parent.
	{
		totalBytelength := 0
		for _, srcloc := range covLoc.SrcLocs {
			totalBytelength += srcloc.ByteLength
		}

		childrenNonEmpty      := 0 < totalBytelength
		childrenDontPartition := totalBytelength < covLoc.CoverageRange.ByteLength
		isLeafNode            := len(node.Children) == 0

		if childrenNonEmpty && (childrenDontPartition || isLeafNode) {
			covLocs = append(covLocs, covLoc)
		}
	}

	sort.Slice(covLocs, func(i, j int) bool {
		if covLocs[i].CoverageRange.ByteOffset == covLocs[j].CoverageRange.ByteOffset {
			// This essentially sorts the parents after the children. This is
			// arbitrary, but needs to be consistently dealt with.
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
