package graph

import (
	"fmt"
)

type watchGraphWebSocketRequest struct {
	Queries    []query     `json:"queries,omitempty"`
	Namespaces *namespaces `json:"namespaces,omitempty"`
	Roots      *roots      `json:"roots,omitempty"`
}

type query struct {
	Include *queryInclude `json:"include,omitempty"`
	Exclude *queryExclude `json:"exclude,omitempty"`
}

type queryInclude struct {
	ResourceSelectorExpression string              `json:"resource_selector_expression,omitempty"`
	Object                     *queryIncludeObject `json:"object,omitempty"`
}

type queryIncludeObject struct {
	LabelSelector            string `json:"label_selector,omitempty"`
	FieldSelector            string `json:"field_selector,omitempty"`
	ObjectSelectorExpression string `json:"object_selector_expression,omitempty"`
	JsonPath                 string `json:"json_path,omitempty"`
}

type queryExclude struct {
	ResourceSelectorExpression string `json:"resource_selector_expression,omitempty"`
}

type namespaces struct {
	Names                    []string `json:"names,omitempty"`
	LabelSelector            string   `json:"label_selector,omitempty"`
	FieldSelector            string   `json:"field_selector,omitempty"`
	ObjectSelectorExpression string   `json:"object_selector_expression,omitempty"`
}

type roots struct {
	Individual []rootsIndividual `json:"individual,omitempty"`
	Selector   []rootsSelector   `json:"selector,omitempty"`
}

type rootsIndividual struct {
	Group     string `json:"group,omitempty"`
	Resource  string `json:"resource,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

type rootsSelector struct {
	Group         string `json:"group,omitempty"`
	Resource      string `json:"resource,omitempty"`
	LabelSelector string `json:"label_selector,omitempty"`
	FieldSelector string `json:"field_selector,omitempty"`
}

type jsonWatchGraphResponse struct {
	Actions  []jsonWatchGraphAction  `json:"actions,omitempty"`
	Warnings []jsonWatchGraphWarning `json:"warnings,omitempty"`
	Error    *jsonWatchGraphError    `json:"error,omitempty"`
}

type jsonWatchGraphAction struct {
	SetVertex    *jsonSetVertex    `json:"svx,omitempty"`
	DeleteVertex *jsonDeleteVertex `json:"dvx,omitempty"`
	SetArc       *jsonSetArc       `json:"sarc,omitempty"`
	DeleteArc    *jsonDeleteArc    `json:"darc,omitempty"`
}

type vertex struct {
	Group     string `json:"g,omitempty"`
	Version   string `json:"v"`
	Resource  string `json:"r"`
	Namespace string `json:"ns,omitempty"`
	Name      string `json:"n"`
}

func (v vertex) String() string {
	return fmt.Sprintf("%s/%s/%s ns=%s n=%s", v.Group, v.Version, v.Resource, v.Namespace, v.Name)
}

func (v *vertex) IsNamespace() bool {
	return v.Group == "" && v.Resource == "namespaces"
}

func (v *vertex) IsNamespaced() bool {
	return v.Namespace != ""
}

type jsonSetVertex struct {
	Vertex   vertex         `json:"vx"`
	Object   map[string]any `json:"o,omitempty"`
	JSONPath []any          `json:"j,omitempty"`
}

type jsonDeleteVertex struct {
	Vertex vertex `json:"vx"`
}

type Arc struct {
	Source      vertex  `json:"s"`
	Destination vertex  `json:"d"`
	Type        arcType `json:"t"`
}

type jsonSetArc struct {
	Arc
	Attributes jsonArcAttrs `json:"a,omitempty"`
}

type jsonArcAttrs struct {
	Controller              bool `json:"c,omitempty"`
	BlockOwnerDeletion      bool `json:"b,omitempty"`
	DestinationDoesNotExist bool `json:"e,omitempty"`
}

type jsonDeleteArc struct {
	Arc
}

type jsonWatchGraphWarning struct {
	Type       string         `json:"t"`
	Message    string         `json:"m"`
	Attributes map[string]any `json:"a,omitempty"`
}

type jsonWatchGraphError struct {
	Code    uint16 `json:"code"`
	CodeStr string `json:"code_string"`
	Reason  string `json:"reason"`
}

func (e *jsonWatchGraphError) Error() string {
	return fmt.Sprintf("%s (%d): %s", e.CodeStr, e.Code, e.Reason)
}
