package v1alpha3

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	v2 "github.com/mbrt/gmailctl/pkg/config/v1alpha2"
)

var dummyFilter = FilterNode{}

// Import converts a v2 config into a v3.
func Import(cfg v2.Config) (Config, error) {
	i := importer{}
	return i.Import(cfg)
}

type importer struct {
	nmap namedFilterMap
	err  error
}

func (i *importer) Import(cfg v2.Config) (Config, error) {
	i.importNamedFilters(cfg.Filters)
	finalErr := i.resetError()

	var rules []Rule
	for _, r := range cfg.Rules {
		rules = append(rules, i.importRule(r))
		if err := i.resetError(); err != nil {
			finalErr = multierror.Append(finalErr,
				errors.Wrapf(err, "invalid rule: %s", r),
			)
		}
	}

	return Config{
		Version: Version,
		Author:  cfg.Author,
		Rules:   rules,
	}, finalErr
}

func (i *importer) importNamedFilters(fs []v2.NamedFilter) {
	i.nmap = namedFilterMap{}
	var finalErr error

	for _, f := range fs {
		i.nmap[f.Name] = i.importFilter(f.Query)
		if err := i.resetError(); err != nil {
			finalErr = multierror.Append(finalErr,
				errors.Wrapf(err, "invalid filter '%s': %s", f.Name, f.Query),
			)
		}
	}

	i.err = finalErr
}

func (i *importer) importRule(r v2.Rule) Rule {
	return Rule{
		Filter:  i.importFilter(r.Filter),
		Actions: r.Actions,
	}
}

func (i *importer) importFilter(f v2.FilterNode) FilterNode {
	if f.RefName != "" {
		return i.importRefName(f.RefName)
	}

	var not *FilterNode
	if f.Not != nil {
		nf := i.importFilter(*f.Not)
		not = &nf
	}
	return FilterNode{
		And:     i.importFilters(f.And),
		Or:      i.importFilters(f.Or),
		Not:     not,
		From:    f.From,
		To:      f.To,
		Cc:      f.Cc,
		Subject: f.Subject,
		List:    f.List,
		Has:     f.Has,
		Query:   f.Query,
	}
}

func (i *importer) importFilters(ns []v2.FilterNode) []FilterNode {
	var res []FilterNode
	for _, f := range ns {
		res = append(res, i.importFilter(f))
	}
	return res
}

func (i *importer) importRefName(name string) FilterNode {
	if n, ok := i.nmap[name]; ok {
		return n
	}
	i.err = multierror.Append(i.err,
		errors.Errorf("filter name '%s' not found", name))
	return dummyFilter
}

func (i *importer) resetError() error {
	err := i.err
	i.err = nil
	return err
}

type namedFilterMap map[string]FilterNode
