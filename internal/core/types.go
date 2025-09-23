package core

import "time"

type Metadata struct {
	Generation        uint32     `json:"generation"`
	Finalizers        []string   `json:"finalizers,omitempty"`
	CreationTimestamp *time.Time `json:"creationTimestamp"`
	DeletionTimestamp *time.Time `json:"deletionTimestamp,omitempty"`
}
