package autoupdate_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/OpenSlides/openslides-autoupdate-service/internal/autoupdate"
	"github.com/OpenSlides/openslides-autoupdate-service/internal/keysbuilder"
	"github.com/OpenSlides/openslides-autoupdate-service/pkg/datastore/dsmock"
	"github.com/OpenSlides/openslides-autoupdate-service/pkg/environment"
)

const dataSet = `---
user/1/id: 1
a/1:
	a: a1
	title: a1
	b_id: 1
	c_ids: []
	ga_ids: [1,2]
a/2:
	a: a2
	title: a2
	c_ids: [1,2]
	ga_ids: []
b/1:
	b: b1
	title: b1
	a_id: 1
	c_ids: [1]
	gb_id: 1
	b_children_ids: [2]
	d_ids: [1]
b/2:
	b: b2
	title: b2
	c_ids: [1,2]
	b_parent_id: 1
	b_children_ids: []
	d_ids: [1,2]
c/1:
	c: c1
	title: c1
	a_id: 2
	b_ids: [1,2]
	ga_ids: [2,3]
c/2:
	c: c2
	title: c2
	a_id: 2
	b_ids: [2]
	ga_ids: [2,3]
d/1:
	d: d1
	b_$_ids: ["1","2","3"]
	b_$1_ids: [1,2]
	b_$2_ids: [1]
	b_$3_ids: []
d/2:
	d: d2
	b_$_ids: ["1","4"]
	b_$1_ids: []
	b_$4_ids: [2]
ga/1:
	ga: ga.1
	content_object_ids: ["a/1"]
ga/2:
	ga: ga.2
	content_object_ids: ["a/1","c/1","c/2"]
ga/3:
	ga: ga.3
	content_object_ids: ["c/1","c/2"]
gb/1:
	gb: gb.1
	content_object_id: b/1
`

func TestFeatures(t *testing.T) {
	datastore, _ := dsmock.NewMockDatastore(dsmock.YAMLData(dataSet))
	lookup := environment.ForTests{}
	service, _, _ := autoupdate.New(lookup, datastore, RestrictAllowed)

	for _, tt := range []struct {
		name string

		// request is an example request to the autoupdate service. The
		// http-server expects not one request, but a list of requests.
		request string

		// result is the data returned for the request. The http-server returns
		// the same json encoded data but its format differs. It only returns
		// one line without spaces.
		result string
	}{
		{
			"Basic",
			`{
				"collection": "a",
				"ids": [
					1,
					2
				],
				"fields": {
					"a": null,
					"c_ids": {
						"type": "relation-list",
						"collection": "c",
						"fields": {
							"c": null,
							"ga_ids": {
								"type": "relation-list",
								"collection": "ga",
								"fields": {
									"ga": null
								}
							}
						}
					},
					"b_id": {
						"type": "relation",
						"collection": "b",
						"fields": {}
					},
					"ga_ids": {
						"type": "relation-list",
						"collection": "ga",
						"fields": {
							"ga": null
						}
					}
				}
			}`,
			`{
				"a/1/a":      "a1",
				"a/1/c_ids":  [],
				"a/1/b_id":   1,
				"a/1/ga_ids": [1,2],
				"a/2/a":      "a2",
				"a/2/c_ids":  [1,2],
				"a/2/ga_ids": [],
				"c/1/c":      "c1",
				"c/1/ga_ids": [2,3],
				"c/2/c":      "c2",
				"c/2/ga_ids": [2,3],
				"ga/1/ga":    "ga.1",
				"ga/2/ga":    "ga.2",
				"ga/3/ga":    "ga.3"
			}`,
		},
		{
			"Partial merged fields, generic lookup",
			`{
				"collection": "gb",
				"ids": [1],
				"fields": {
					"content_object_id": {
						"type": "generic-relation",
						"fields": {
							"b_children_ids": {
								"type": "relation-list",
								"collection": "b",
								"fields": {
									"c_ids": {
										"type": "relation-list",
										"collection": "c",
										"fields": {
											"c": null
										}
									},
									"b_parent_id": null
								}
							},
							"c_ids": {
								"type": "relation-list",
								"collection": "c",
								"fields": {
									"c": null,
									"title": null
								}
							},
							"gb_id": null
						}
					}
				}
			}`,
			`{
				"b/1/b_children_ids":     [2],
				"b/1/c_ids":              [1],
				"b/1/gb_id":              1,
				"b/2/c_ids":              [1,2],
				"b/2/b_parent_id":        1,
				"gb/1/content_object_id": "b/1",
				"c/1/c":                  "c1",
				"c/1/title":              "c1",
				"c/2/c":                  "c2"
			}`,
		},
		{
			"non-existent ids, fields, fqids, references, generic relations and fields without a relation",
			`{
				"collection": "ga",
				"ids": [2,4],
				"fields": {
					"content_object_ids": {
						"type": "generic-relation-list",
						"fields": {
							"a": null,
							"b": null,
							"not_existent": {
								"type": "generic-relation",
								"fields": {"key": null}
							},
							"title": null,
							"ga_ids": null,
							"a_id": null
						}
					}
				}
			}`,
			`{
				"ga/2/content_object_ids": ["a/1","c/1","c/2"],
				"a/1/a":                   "a1",
				"a/1/title":               "a1",
				"a/1/ga_ids":              [1,2],
				"c/1/title":               "c1",
				"c/1/a_id":                2,
				"c/1/ga_ids":              [2,3],
				"c/2/title":               "c2",
				"c/2/a_id":                2,
				"c/2/ga_ids":              [2,3]
			}`,
		},
		{
			"template fields",
			`{
				"collection": "d",
				"ids": [1,2],
				"fields": {
					"d": null,
					"b_$_ids": null
				}
			}`,
			`{
				"d/1/d":       "d1",
				"d/1/b_$_ids": ["1","2","3"],
				"d/2/d":       "d2",
				"d/2/b_$_ids": ["1","4"]
			}`,
		},
		{
			"structured fields without references",
			`{
				"collection": "d",
				"ids": [1,2],
				"fields": {
					"d": null,
					"b_$_ids": {
						"type": "template"
					}
				}
			}`,
			`{
				"d/1/d":       "d1",
				"d/1/b_$_ids": ["1","2","3"],
				"d/1/b_$1_ids": [1,2],
				"d/1/b_$2_ids": [1],
				"d/1/b_$3_ids": [],
				"d/2/d":       "d2",
				"d/2/b_$_ids": ["1","4"],
				"d/2/b_$1_ids": [],
				"d/2/b_$4_ids": [2]
			}`,
		},
		{
			"structed references",
			`{
				"collection": "d",
				"ids": [1,2],
				"fields": {
					"b_$_ids": {
						"type": "template",
						"values": {
							"type": "relation-list",
							"collection": "b",
							"fields": {
								"b": null
							}
						}
					},
					"b_$4_ids": {
						"type": "relation-list",
						"collection": "b",
						"fields": {
							"title": null
						}
					}
				}
			}`,
			`{
				"d/1/b_$_ids": ["1","2","3"],
				"d/1/b_$1_ids": [1,2],
				"d/1/b_$2_ids": [1],
				"d/1/b_$3_ids": [],
				"d/2/b_$_ids": ["1","4"],
				"d/2/b_$1_ids": [],
				"d/2/b_$4_ids": [2],
				"b/1/b":       "b1",
				"b/2/b":       "b2",
				"b/2/title":   "b2"
			}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := keysbuilder.FromJSON(strings.NewReader(tt.request))
			if err != nil {
				t.Fatalf("FromJSON() returned an unexpected error: %v", err)
			}

			conn, err := service.Connect(context.Background(), 1, builder)
			if err != nil {
				t.Fatalf("creating conection: %v", err)
			}
			next, _ := conn()

			data, err := next(context.Background())
			if err != nil {
				t.Fatalf("Can not get data: %v", err)
			}

			converted := make(map[string]json.RawMessage, len(data))
			for k, v := range data {
				converted[k.String()] = v
			}

			var expect map[string]json.RawMessage
			if err := json.Unmarshal([]byte(tt.result), &expect); err != nil {
				t.Fatalf("Can not decode keys from expected data: %v", err)
			}

			cmpMap(t, converted, expect)
		})
	}
}

func cmpMap(t *testing.T, got, expect map[string]json.RawMessage) {
	t.Helper()
	v1, _ := json.Marshal(got)
	v2, _ := json.Marshal(expect)
	if string(v1) != string(v2) {
		t.Errorf("got %s, expected %s", string(v1), string(v2))
	}
}
