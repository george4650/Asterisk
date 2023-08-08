package models

type File struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Changed string `json:"changed"`
	Size    string `json:"size"`
}
