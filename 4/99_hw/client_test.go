package main

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

type Person struct {
	ID            int    `xml:"id"`
	Guid          string `xml:"guid"`
	IsActive      string `xml:"isActive"`
	Balance       string `xml:"balance"`
	Picture       string `xml:"picture"`
	Age           int    `xml:"age"`
	EyeColor      string `xml:"eyeColor"`
	FirstName     string `xml:"first_name"`
	LastName      string `xml:"last_name"`
	Gender        string `xml:"gender"`
	Company       string `xml:"company"`
	Email         string `xml:"email"`
	Phone         string `xml:"phone"`
	Address       string `xml:"address"`
	About         string `xml:"about"`
	Registered    string `xml:"registered"`
	FavoriteFruit string `xml:"favoriteFruit"`
}

type Root struct {
	XMLName xml.Name `xml:"root"`
	Persons []Person `xml:"row"`
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	var sr SearchRequest
	var err error

	sr.Query = r.FormValue("query")
	sr.OrderField = r.FormValue("order_field")

	if orderByValue := r.FormValue("order_by"); orderByValue != "" {
		sr.OrderBy, err = strconv.Atoi(orderByValue)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Invalid order_by value"}`))
			return
		}
	}

	if offsetValue := r.FormValue("offset"); offsetValue != "" {
		sr.Offset, err = strconv.Atoi(offsetValue)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Invalid offset value"}`))
			return
		}
	}

	if limitValue := r.FormValue("limit"); limitValue != "" {
		sr.Limit, err = strconv.Atoi(limitValue)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Invalid limit value"}`))
			return
		}
	}

	f, err := os.Open("dataset.xml")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Cannot open dataset"}`))
		return
	}

	decoder := xml.NewDecoder(f)
	root := &Root{}

	if err := decoder.Decode(root); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Cannot decode dataset"}`))
		return
	}

	var filteredUsers []User
	for _, person := range root.Persons {
		name := person.FirstName + " " + person.LastName
		if strings.Contains(name, sr.Query) || strings.Contains(person.About, sr.Query) {
			user := User{
				Id:     person.ID,
				Name:   name,
				Age:    person.Age,
				About:  strings.TrimSpace(person.About),
				Gender: person.Gender,
			}
			filteredUsers = append(filteredUsers, user)
		}
	}

	if sr.OrderBy != OrderByAsIs && sr.OrderBy != OrderByDesc && sr.OrderBy != OrderByAsc {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid order_by value"}`))
		return
	}

	if sr.OrderField != "Id" && sr.OrderField != "Age" && sr.OrderField != "Name" && sr.OrderField != "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid order_field value"}`))
		return
	}

	if sr.OrderBy != OrderByAsIs {
		slices.SortFunc(filteredUsers, func(a, b User) int {
			switch sr.OrderField {
			case "Id":
				if sr.OrderBy == OrderByAsc {
					return a.Id - b.Id
				}
				if sr.OrderBy == OrderByDesc {
					return b.Id - a.Id
				}
			case "Age":
				if sr.OrderBy == OrderByAsc {
					return a.Age - b.Age
				}
				if sr.OrderBy == OrderByDesc {
					return b.Age - a.Age
				}
			case "Name", "":
				if sr.OrderBy == OrderByAsc {
					return strings.Compare(a.Name, b.Name)
				}
				if sr.OrderBy == OrderByDesc {
					return strings.Compare(b.Name, a.Name)
				}
			}
			return 0
		})
	}

	if sr.Offset < 0 || sr.Offset > len(filteredUsers) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid offset value"}`))
		return
	}
	filteredUsers = filteredUsers[sr.Offset:]

	if sr.Limit < 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid limit  value"}`))
		return
	}

	if sr.Limit != 0 && sr.Limit <= len(filteredUsers) {
		filteredUsers = filteredUsers[:sr.Limit]
	}

	jsonPersons, err := json.Marshal(filteredUsers)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Failed to convert users to json"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonPersons)
}

type TestCase struct {
	sRequest  SearchRequest
	sResponse *SearchResponse
	err       error
}

func TestSearchServer(t *testing.T) {
	cases := []TestCase{
		TestCase{
			sRequest: SearchRequest{
				Limit:      1,
				Offset:     0,
				Query:      "",
				OrderField: "",
				OrderBy:    0,
			},
			sResponse: &SearchResponse{
				Users: []User{
					{
						Id:     0,
						Name:   "Boyd Wolf",
						Age:    22,
						About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.",
						Gender: "male",
					},
				},
				NextPage: true,
			},
			err: nil,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(SearchServer))

	for i, c := range cases {
		client := &SearchClient{URL: server.URL}
		result, err := client.FindUsers(c.sRequest)

		if (err != nil) != (c.err != nil) {
			t.Errorf("[%d] unexpected error: %v, expected: %v", i, err, c.err)
		}

		if !reflect.DeepEqual(result, c.sResponse) {
			t.Errorf("[%d] wrong result:\n %#v\n expected:\n %#v", i, result, c.sResponse)
		}

	}

}
