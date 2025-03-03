package keysbuilder_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/OpenSlides/openslides-autoupdate-service/internal/keysbuilder"
)

func TestJSONValid(t *testing.T) {
	json := strings.NewReader(`
	{
		"ids": [5],
		"collection": "user",
		"fields": {
			"motion_ids": {
				"type": "relation-list",
				"collection": "motion",
				"fields": {"name": null}
			}
		}
	}
	`)
	if _, err := keysbuilder.FromJSON(json); err != nil {
		t.Errorf("Got unexpected error: %v", err)
	}
}

func TestJSONInvalid(t *testing.T) {
	json := strings.NewReader(`{5`)
	_, err := keysbuilder.FromJSON(json)
	if err == nil {
		t.Errorf("FromJSON did not return an error")
	}
	var errJSON keysbuilder.JSONError
	if !errors.As(err, &errJSON) {
		t.Errorf("Got error %v, expected error to be of type ErrJSON", err)
	}
}

func TestJSONIDNull(t *testing.T) {
	json := strings.NewReader(`
	{
		"ids": [null],
		"collection": "user",
		"fields": {"name": null}
	}
	`)
	_, err := keysbuilder.FromJSON(json)
	if err == nil {
		t.Errorf("Expected an error, got none")
	}
	var errInvalid keysbuilder.InvalidError
	if !errors.As(err, &errInvalid) {
		t.Errorf("Expected error to be of type ErrJSON, got: %v", err)
	}
}

func TestJSONSingleID(t *testing.T) {
	json := strings.NewReader(`
	{
		"ids": 5,
		"collection": "user",
		"fields": {"name": null}
	}
	`)
	_, err := keysbuilder.FromJSON(json)
	if err == nil {
		t.Errorf("Expected an error, got none")
	}
	var errJSON keysbuilder.JSONError
	if !errors.As(err, &errJSON) {
		t.Errorf("Expected error to be of type ErrJSON, got: %v", err)
	}
}

func TestJSONSuffixNoFields(t *testing.T) {
	json := strings.NewReader(`
	{
		"ids": [5],
		"collection": "user",
		"fields": {
			"group_ids": null,
			"note_id": null
		}
	}
	`)
	_, err := keysbuilder.FromJSON(json)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestRequestErrors(t *testing.T) {
	for _, tt := range []struct {
		name   string
		input  string
		msg    string
		fields []string
	}{
		{
			"No Collection",
			`{
				"ids": [5],
				"fields": {"name": null}
			}`,
			"attribute collection is missing",
			nil,
		},
		{
			"no ids",
			`{
				"fields": {"name": null},
				"collection": "user"
			}`,
			"no ids",
			nil,
		},
		{
			"Relation no collection",
			`{
				"ids": [5],
				"collection": "user",
				"fields": {
					"group_id": {
						"type": "relation",
						"fields": {"name": null}
					}
				}
			}`,
			`field "group_id": no collection`,
			[]string{"group_id"},
		},
		{
			"NoType",
			`{
				"ids": [5],
				"collection": "user",
				"fields": {
					"group_id": {
						"collection": "group",
						"fields": {"name": null}
					}
				}
			}`,
			`field "group_id": no type`,
			[]string{"group_id"},
		},
		{
			"NoType sub",
			`	{
				"ids": [5],
				"collection": "user",
				"fields": {
					"group_id": {
						"type": "relation-list",
						"collection": "group",
						"fields": {
							"perm_ids": {
								"fields": {
									"collection": "perm",
									"fields": {"name": null}
								}
							}
						}
					}
				}
			}`,
			`field "group_id.perm_ids": no type`,
			[]string{"group_id", "perm_ids"},
		},
		{
			"NoType sub",
			`	{
				"ids": [5],
				"collection": "user",
				"fields": {
					"group_id": {
						"type": "relation-list",
						"collection": "group",
						"fields": {
							"perm_ids": {
								"fields": {
									"collection": "perm",
									"fields": {"name": null}
								}
							}
						}
					}
				}
			}`,
			`field "group_id.perm_ids": no type`,
			[]string{"group_id", "perm_ids"},
		},
		{
			"Unknown Type",
			`	{
				"ids": [5],
				"collection": "user",
				"fields": {
					"group_id": {
						"type": "invalid-type",
						"collection": "group",
						"fields": {"name": null}
					}
				}
			}`,
			`field "group_id": unknown type invalid-type`,
			[]string{"group_id"},
		},
		{
			"Relation twice no fields",
			`{
				"ids": [5],
				"collection": "user",
				"fields": {
					"group_ids": {
						"type": "relation-list",
						"collection": "group",
						"fields": {
							"perm_ids": {
								"type": "relation-list",
								"collection": "perm"
							}
						}
					}
				}
			}`,
			`field "group_ids.perm_ids": no fields`,
			[]string{"group_ids", "perm_ids"},
		},
		{
			"collection has upper letter",
			`{
				"ids": [1],
				"collection": "User",
				"fields": {"username": null}
			}
			`,
			"invalid collection name",
			nil,
		},
		{
			"field with upper letter",
			`{
				"ids": [1],
				"collection": "user",
				"fields": {"Username": null}
			}
			`,
			"field \"Username\": fieldname \"Username\" is not a valid fieldname",
			[]string{"Username"},
		},
		{
			"collection in relation-field has upper letter",
			`{
				"ids": [1],
				"collection": "user",
				"fields": {
					"group_id": {
						"type": "relation",
						"collection": "Group",
						"fields": {"name": null}
					}
				}
			}
			`,
			"field \"group_id\": invalid collection name",
			[]string{"group_id"},
		},
		{
			"collection in relation-list-field has upper letter",
			`{
				"ids": [1],
				"collection": "user",
				"fields": {
					"group_ids": {
						"type": "relation-list",
						"collection": "Group",
						"fields": {"name": null}
					}
				}
			}
			`,
			"field \"group_ids\": invalid collection name",
			[]string{"group_ids"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := keysbuilder.FromJSON(strings.NewReader(tt.input))
			if err == nil {
				t.Errorf("Expected an error, got none")
			}
			var kErr keysbuilder.InvalidError
			if !errors.As(err, &kErr) {
				t.Errorf("Expected err to be %T, got: %v", kErr, err)
			}
			if got := kErr.Error(); got != tt.msg {
				t.Errorf("Expected error message %q, got: %q", tt.msg, got)
			}
			if fields := kErr.Fields(); !cmpSlice(fields, tt.fields) {
				t.Errorf("Expected error to be on field \"%v\", got %v", tt.fields, fields)
			}
		})
	}
}

func TestManyFromJSON(t *testing.T) {
	json := strings.NewReader(`[
	{
		"ids": [5],
		"collection": "user",
		"fields": {
			"group_ids": {
				"type": "relation-list",
				"collection": "group",
				"fields": {
					"name": null
				}
			}
		}
	},
	{
		"ids": [5],
		"collection": "user",
		"fields": {
			"name": null
		}
	}]`)

	_, err := keysbuilder.ManyFromJSON(json)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestManyFromJSONInvalidJSON(t *testing.T) {
	json := strings.NewReader(`[
	{
		"ids": [5],
		"collection": "user",
		"fields": {
			"group_ids": {
				"type": "relation-list",
				"collection": "group",
				"fields": {
					"name": null
				}
			}
		}
	},
	{
		"ids": [5],
		"collection": "user",
		"fi
	}]`)
	_, err := keysbuilder.ManyFromJSON(json)
	if err == nil {
		t.Error("Expected ManyFromJSON() to return an error, got not")
	}
	var errJSON keysbuilder.JSONError
	if !errors.As(err, &errJSON) {
		t.Errorf("Expected error to be of type ErrJSON, got: %v", err)
	}
}

func TestManyFromJSONInvalidInput(t *testing.T) {
	json := strings.NewReader(`[
	{
		"ids": [5],
		"collection": "user",
		"fields": {
			"group_ids": {
				"type": "relation-list",
				"collection": "group",
				"fields": {
					"name": null
				}
			}
		}
	},
	{
		"ids": [5],
		"collection": "user",
		"fields": {
			"group_ids": {
				"type": "relation-list",
				"fields": {
					"name": null
				}
			}
		}
	}]`)
	_, err := keysbuilder.ManyFromJSON(json)
	if err == nil {
		t.Error("Expected ManyFromJSON() to return an error, got not")
	}
	var kErr keysbuilder.InvalidError
	if !errors.As(err, &kErr) {
		t.Errorf("Expected error to be of type ErrInvalid, got: %v", err)
	}
}
