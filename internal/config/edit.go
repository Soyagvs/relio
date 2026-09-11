package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Field is one scalar path/value pair for SetFields. Path is a sequence of
// mapping keys (e.g. {"release", "contributors"}); Value is encoded with
// (*yaml.Node).Encode, so bool, string, and int all work.
type Field struct {
	Path  []string
	Value any
}

// SetFields updates root/.release.yaml in place, setting exactly the given
// scalar paths and creating any missing intermediate mappings. Every other
// key, every comment, and the key order are preserved — this is a surgical
// node-level edit, never a full yaml.Marshal(Config) rewrite.
//
// A full rewrite is not just a formatting concern: ReleaseConfig.Changelog
// and .Tag are *bool with nil-means-default semantics (see ChangelogEnabled /
// TagEnabled), so re-marshaling the whole Config would turn an omitted
// `changelog:`/`tag:` key into an explicit `changelog: null`/`tag: null`, and
// would materialize empty `github:`/`content:` blocks that were absent from
// the source file. SetFields never touches a key it was not asked to set.
//
// SetFields returns ErrNotFound when no config file exists at root, mirroring
// Load's behavior for the same case: SetFields modifies an existing config,
// it does not create one. A caller that wants create-on-first-write calls
// Save first.
//
// One accepted, documented tradeoff: because the whole document is
// re-encoded on write, indentation is normalized to 4 spaces (matching
// Save's yaml.Marshal default) even though keys, order, and comments survive
// untouched. A hand-written 2-space file is reindented on its first
// SetFields call.
func SetFields(root string, fields ...Field) error {
	p := Path(root)
	info, err := os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("config: parsing %s: %w", FileName, err)
	}
	if doc.Kind == 0 || len(doc.Content) == 0 {
		// Empty file: start from an empty mapping document.
		doc = yaml.Node{
			Kind:    yaml.DocumentNode,
			Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}},
		}
	}

	m := doc.Content[0]
	if m.Kind == yaml.AliasNode {
		return fmt.Errorf("config: %s has an aliased document root, which SetFields does not support", FileName)
	}
	if m.Kind != yaml.MappingNode {
		return fmt.Errorf("config: %s is not a mapping", FileName)
	}

	for _, f := range fields {
		if err := setField(m, f); err != nil {
			return err
		}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(4)
	if err := enc.Encode(&doc); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	return writeFileAtomic(p, buf.Bytes(), info.Mode())
}

// setField sets a single Field within root, creating missing intermediate
// mappings and overwriting a leaf's Kind/Tag/Value/Style in place so its
// HeadComment/LineComment/FootComment/Anchor survive.
func setField(root *yaml.Node, f Field) error {
	if len(f.Path) == 0 {
		return fmt.Errorf("config: SetFields: empty field path")
	}

	cur := root
	for _, key := range f.Path[:len(f.Path)-1] {
		child := mapValue(cur, key)
		switch {
		case child == nil:
			child = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			appendPair(cur, key, child)
		case child.Kind == yaml.ScalarNode && child.Tag == "!!null":
			child.Kind = yaml.MappingNode
			child.Tag = "!!map"
			child.Value = ""
			child.Style = 0
			child.Content = nil
		case child.Kind != yaml.MappingNode:
			return fmt.Errorf("config: %s is not a mapping", key)
		}
		cur = child
	}

	last := f.Path[len(f.Path)-1]
	var enc yaml.Node
	if err := enc.Encode(f.Value); err != nil {
		return fmt.Errorf("config: encoding %s: %w", last, err)
	}

	if leaf := mapValue(cur, last); leaf != nil {
		leaf.Kind = enc.Kind
		leaf.Tag = enc.Tag
		leaf.Value = enc.Value
		leaf.Style = enc.Style
		leaf.Content = nil
	} else {
		appendPair(cur, last, &enc)
	}
	return nil
}

// mapValue returns the value node for key within mapping node m, or nil if
// key is not present.
func mapValue(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// appendPair appends a new key/value pair at the end of mapping node m.
func appendPair(m *yaml.Node, key string, val *yaml.Node) {
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, val)
}

// writeFileAtomic writes data to path via a temp file in the same directory
// followed by os.Rename, so an interrupted write cannot truncate the config.
// The temp file is chmod'd to mode before the rename so the final file keeps
// the original permission bits.
func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".release-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename below succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return nil
}
