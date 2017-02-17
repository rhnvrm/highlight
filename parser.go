package highlight

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"regexp"
)

// A Def is a full syntax definition for a language
// It has a filetype, information about how to detect the filetype based
// on filename or header (the first line of the file)
// Then it has the rules which define how to highlight the file
type Def struct {
	ft       string
	ftdetect []*regexp.Regexp
	rules    *Rules
}

// A Pattern is one simple syntax rule
// It has a group that the rule belongs to, as well as
// the regular expression to match the pattern
type Pattern struct {
	group string
	regex *regexp.Regexp
}

// Rules defines which patterns and regions can be used to highlight
// a filetype
type Rules struct {
	regions  []*Region
	patterns []*Pattern
}

// A Region is a highlighted region (such as a multiline comment, or a string)
// It belongs to a group, and has start and end regular expressions
// A Region also has rules of its own that only apply when matching inside the
// region and also rules from the above region do not match inside this region
// Note that a region may contain more regions
type Region struct {
	group  string
	parent *Region
	start  *regexp.Regexp
	end    *regexp.Regexp
	rules  *Rules
}

// ParseDef parses an input syntax file into a highlight Def
func ParseDef(input []byte) (*Def, error) {
	var rules map[interface{}]interface{}
	if err := yaml.Unmarshal(input, &rules); err != nil {
		return nil, err
	}

	s := new(Def)

	for k, v := range rules {
		if k == "filetype" {
			filetype := v.(string)

			s.ft = filetype
		} else if k == "detect" {
			ftdetect := v.(map[interface{}]interface{})
			if len(ftdetect) >= 1 {
				syntax, err := regexp.Compile(ftdetect["filename"].(string))
				if err != nil {
					return nil, err
				}

				s.ftdetect = append(s.ftdetect, syntax)
			}
			if len(ftdetect) >= 2 {
				header, err := regexp.Compile(ftdetect["header"].(string))
				if err != nil {
					return nil, err
				}

				s.ftdetect = append(s.ftdetect, header)
			}
		} else if k == "rules" {
			inputRules := v.([]interface{})

			rules, err := parseRules(inputRules, nil)
			if err != nil {
				return nil, err
			}

			s.rules = rules
		}
	}

	return s, nil
}

func parseRules(input []interface{}, curRegion *Region) (*Rules, error) {
	rules := new(Rules)

	for _, v := range input {
		rule := v.(map[interface{}]interface{})
		for k, val := range rule {
			group := k

			switch object := val.(type) {
			case string:
				// Pattern
				r, err := regexp.Compile(object)
				if err != nil {
					return nil, err
				}

				rules.patterns = append(rules.patterns, &Pattern{group.(string), r})
			case map[interface{}]interface{}:
				// Region
				region, err := parseRegion(group.(string), object, curRegion)
				if err != nil {
					return nil, err
				}
				rules.regions = append(rules.regions, region)
			default:
				return nil, fmt.Errorf("Bad type %T", object)
			}
		}
	}

	return rules, nil
}

func parseRegion(group string, regionInfo map[interface{}]interface{}, prevRegion *Region) (*Region, error) {
	var err error

	region := new(Region)
	region.group = group
	region.parent = prevRegion

	region.start, err = regexp.Compile(regionInfo["start"].(string))

	if err != nil {
		return nil, err
	}

	region.end, err = regexp.Compile(regionInfo["end"].(string))

	if err != nil {
		return nil, err
	}

	region.rules, err = parseRules(regionInfo["rules"].([]interface{}), region)

	if err != nil {
		return nil, err
	}

	return region, nil
}
