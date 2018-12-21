package ast

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/reserve-protocol/solstice/common"
	"github.com/reserve-protocol/solstice/solc"
	"github.com/reserve-protocol/solstice/srclocation"
	"github.com/reserve-protocol/solstice/srcmap"
)

// The Abstract Syntax Tree of a contract.
type AST struct {
	ID       uint
	SrcLoc   srclocation.SourceLocation
	Children []*AST
}

func Get(contractName string) (AST, error) {
	srcMapJSON, err := solc.GetCombinedJSON("ast", []string{contractName})
	if err != nil {
		return AST{}, err
	}

	return processASTNode(
		srcMapJSON.Sources[contractName].AST,
		srcMapJSON.SourceList,
	)
}

// Convert tree from solc's raw string & int representation to our SourceLocation type
func processASTNode(node solc.JSONAST, sourceList []string) (AST, error) {
	var newTree AST
	newTree.ID = node.ID

	srcLocParts := strings.Split(node.Src, ":")

	byteOffset, err := strconv.Atoi(srcLocParts[0])
	if err != nil {
		return newTree, err
	}

	byteLength, err := strconv.Atoi(srcLocParts[1])
	if err != nil {
		return newTree, err
	}

	sourceFileIndex, err := strconv.Atoi(srcLocParts[2])
	if err != nil {
		return newTree, err
	}

	newTree.SrcLoc = srclocation.SourceLocation{
		byteOffset,
		byteLength,
		sourceList[sourceFileIndex],
		rune(0),
	}

	for _, childNode := range node.Children {
		newNode, err := processASTNode(*childNode, sourceList)
		if err != nil {
			return newTree, err
		}
		newTree.Children = append(newTree.Children, &newNode)
	}

	return newTree, nil
}

// The AST from a source map will contain only those byte ranges that are
// represented in the bytecode, since the source map comes from the bytecode.
func FromSrcmaps() (map[string]AST, error) {
	treesFromSrcmaps := make(map[string]AST)

	srcMaps, _, err := srcmap.Get()
	if err != nil {
		return treesFromSrcmaps, err
	}

	contracts, err := common.AllContracts()
	if err != nil {
		return treesFromSrcmaps, err
	}

	// You want to make top level srclocs for each tree, in case the srcmap has several top-level
	// srcloc nodes, like if the file has two contracts
	for _, name := range contracts {
		fileInfo, err := os.Stat(name);
		if err != nil {
		    return treesFromSrcmaps, err
		}

		treesFromSrcmaps[name] = AST{
			SrcLoc: srclocation.SourceLocation{
				ByteOffset:     0,
				ByteLength:     int(fileInfo.Size()),
				SourceFileName: name,
			},
		}
	}

	for _, srcMap := range srcMaps {
		for _, srcLoc := range srcMap {
			if srcLoc.SourceFileName == "" {
				continue
			}
			treeFromSrcmap := treesFromSrcmaps[srcLoc.SourceFileName]
			err = treeFromSrcmap.insertSrcLoc(srcLoc)
			if err != nil {
			    return treesFromSrcmaps, err
			}
			treesFromSrcmaps[srcLoc.SourceFileName] = treeFromSrcmap
		}
	}
	return treesFromSrcmaps, nil
}

// You have a tree and a new node; figure out where inside that tree this new node goes.
func (tree *AST) insertSrcLoc(srcLoc srclocation.SourceLocation) error {
	if srcLoc.ByteOffset == tree.SrcLoc.ByteOffset && srcLoc.ByteLength == tree.SrcLoc.ByteLength {
		// It's redundant for our purposes, and we don't need to insert it.
		return nil
	} else if srcLoc.Overlaps(tree.SrcLoc) {
		var treeCopy AST
		treeCopy = *tree
		tree.SrcLoc = srcLoc
		tree.Children = []*AST{&treeCopy}
	} else if tree.SrcLoc.Overlaps(srcLoc) {
		var treeChildrenUpdated []*AST
		newTree := AST{SrcLoc: srcLoc}
		for _, child := range tree.Children {
			if child.SrcLoc.Overlaps(srcLoc) {
				// Since the children can't overlap, we've found the only child
				// which the srcLoc goes under. We don't have to process any
				// more children or change tree.Children.
				child.insertSrcLoc(srcLoc)
				return nil
			} else if tree.SrcLoc.PartiallyOverlaps(srcLoc) {
				return errors.New("SourceLocation range partially overlaps with a previous range and cannot be inserted into AST.")
			} else if srcLoc.Overlaps(child.SrcLoc) {
				// Don't append child to treeChildrenUpdated
				newTree.Children = append(newTree.Children, child)
				// Don't return because we might want to move more of tree.Children into newTree.Children
			} else if srcLoc.IsDisjointWith(child.SrcLoc) {
				// The child should remain a child of tree, and we should continue to check the rest of the children.
				treeChildrenUpdated = append(treeChildrenUpdated, child)
			} else {
				panic("This is logically impossible.")
			}
		}
		// If we've made it here without returning, then srcLoc does not
		// belong under one of the children and should be in tree.Children.
		tree.Children = append(treeChildrenUpdated, &newTree)
	} else if srcLoc.IsDisjointWith(tree.SrcLoc) {
		return errors.New(fmt.Sprintf("SourceLocation range %v is disjoint with AST.SrcLoc %v and cannot be inserted.", srcLoc, tree.SrcLoc))
	} else {
		return errors.New("SourceLocation range partially overlaps with a previous range and cannot be inserted into AST.")
	}
	return nil
}
