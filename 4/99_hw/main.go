package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
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

	sr.Query = r.FormValue("Query")
	sr.OrderField = r.FormValue("OrderField")

	if orderByValue := r.FormValue("OrderBy"); orderByValue != "" {
		sr.OrderBy, err = strconv.Atoi(orderByValue)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Invalid orderBy value"}`))
			return
		}
	}

	if offsetValue := r.FormValue("Offset"); offsetValue != "" {
		sr.Offset, err = strconv.Atoi(offsetValue)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Invalid offset value"}`))
			return
		}
	}

	if limitValue := r.FormValue("Limit"); limitValue != "" {
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

	var filteredPersons []Person

	for _, person := range root.Persons {
		name := person.FirstName + person.LastName
		if strings.Contains(name, sr.Query) || strings.Contains(person.About, sr.Query) {
			filteredPersons = append(filteredPersons, person)
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
		slices.SortFunc(filteredPersons, func(a, b Person) int {
			switch sr.OrderField {
			case "Id":
				if sr.OrderBy == OrderByAsc {
					return a.ID - b.ID
				}
				if sr.OrderBy == OrderByDesc {
					return b.ID - a.ID
				}
			case "Age":
				if sr.OrderBy == OrderByAsc {
					return a.Age - b.Age
				}
				if sr.OrderBy == OrderByDesc {
					return b.Age - a.Age
				}
			case "Name", "":
				aName := a.FirstName + a.LastName
				bName := b.FirstName + b.LastName
				if sr.OrderBy == OrderByAsc {
					return strings.Compare(aName, bName)
				}
				if sr.OrderBy == OrderByDesc {
					return strings.Compare(bName, aName)
				}
			}
			return 0
		})
	}

	if sr.Offset < 0 || sr.Offset > len(filteredPersons) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid offset value"}`))
		return
	}
	filteredPersons = filteredPersons[sr.Offset:]

	if sr.Limit < 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid limit  value"}`))
		return
	}

	if sr.Limit != 0 && sr.Limit <= len(filteredPersons) {
		filteredPersons = filteredPersons[:sr.Limit]
	}

	filteredData := &Root{}
	filteredData.XMLName = root.XMLName
	filteredData.Persons = filteredPersons

	xmlData, err := xml.MarshalIndent(filteredData, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Failed to encode response"}`))
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(xml.Header))
	w.Write(xmlData)

}

func main() {
	http.HandleFunc("/", SearchServer)
	fmt.Println("Listening.....")
	http.ListenAndServe(":8080", nil)
}
