package cproxy

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"regexp"
)

type Rule struct {
	Host      string `yaml:"host"`
	Regex     string `yaml:"regex"`
	Option    string `yaml:"option"`
	Content   string `yaml:"content"`
	UriRegexp *regexp.Regexp
}

var (
	OPT_TO_STDOUT          = "to-stdout"
	OPT_USE_LOCAL_RESPONSE = "use-local-response"
	OPT_TO_REDIS           = "to-redis"
)

func loadRules(filePath string) ([]Rule, error) {
	cfg, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var out struct {
		Version string `yaml:"version"`
		Rules   []Rule `yaml:"rules"`
	}

	if err = yaml.Unmarshal(cfg, &out); err != nil {
		return nil, err
	}
	return out.Rules, nil
}

type RuleOperator struct {
	Enable bool
	rules  map[string][]Rule
	filter *regexp.Regexp
}

func (r *RuleOperator) Match(host, uri string) (rule Rule, matched bool) {
	rule = Rule{
		Option: OPT_TO_STDOUT,
	}
	matched = false
	if r.filter != nil {
		return rule, r.filter.MatchString(uri)
	}
	if len(r.rules) == 0 {
		return
	}
	rules, ok := r.rules[host]
	if ok {
		for _, rule := range rules {
			if rule.UriRegexp.MatchString(uri) {
				return rule, true
			}
		}
		return
	}
	rules, ok = r.rules["default"]
	if !ok {
		return
	}
	for _, rule := range rules {
		if rule.UriRegexp.MatchString(uri) {
			return rule, true
		}
	}
	return
}

func NewRuleOperator(filePath, filter string) RuleOperator {
	ruleInc := RuleOperator{filter: nil, Enable: false}

	if len(filter) > 0 {
		if f, err := regexp.Compile(filter); err == nil {
			ruleInc.filter = f
			ruleInc.Enable = true
			return ruleInc
		}
	}
	ruleMap := make(map[string][]Rule)
	if rls, err := loadRules(filePath); err != nil {
		return ruleInc
	} else {
		for _, v := range rls {
			v.UriRegexp, err = regexp.Compile(v.Regex)
			if err != nil {
				log.Println("compile error:", err)
			}
			r, ok := ruleMap[v.Host]
			if ok {
				r = append(r, v)
			} else {
				r = []Rule{v,}
			}
			ruleMap[v.Host] = r
			ruleInc.Enable = true
		}
	}
	ruleInc.rules = ruleMap
	return ruleInc
}
