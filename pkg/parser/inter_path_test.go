package parser

import "testing"

func TestInferURLPath(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		params   []Param
		expected string
	}{
		{
			name:     "Get resource by ID",
			method:   "GetUserByID",
			params:   []Param{{Name: "id"}},
			expected: "/user/{id}",
		},
		{
			name:     "List resources",
			method:   "ListUsers",
			params:   nil,
			expected: "/users",
		},
		{
			name:     "Create resource",
			method:   "CreateUser",
			params:   []Param{{Name: "user"}},
			expected: "/users",
		},
		{
			name:     "Update resource by ID",
			method:   "UpdateUser",
			params:   []Param{{Name: "id"}},
			expected: "/user/{id}",
		},
		{
			name:     "Delete resource by ID",
			method:   "DeleteUser",
			params:   []Param{{Name: "id"}},
			expected: "/user/{id}",
		},
		{
			name:     "Get resource by multiple IDs",
			method:   "GetOrderByUserIDAndOrderID",
			params:   []Param{{Name: "userID"}, {Name: "orderID"}},
			expected: "/order/{userid}/{orderid}",
		},
		{
			name:     "Action without params",
			method:   "ListOrders",
			params:   nil,
			expected: "/orders",
		},
		{
			name:     "Unknown method pattern",
			method:   "DoSomethingRandom",
			params:   nil,
			expected: "/do/something/random",
		},
		{
			name:     "Get nested resource",
			method:   "GetCommentsByPostID",
			params:   []Param{{Name: "postID"}},
			expected: "/comments/{postid}",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := inferURLPath("", test.method, test.params...)
			if got != test.expected {
				t.Errorf("inferURLPath(%q, %v) = %q; want %q", test.method, test.params, got, test.expected)
			}
		})
	}
}
