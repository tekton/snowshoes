package lib

import (
	"time"
	"encoding/json"
)

type ReqRtn struct {
	Code int    `json:"code"`
	Type string `json:"type"`
	Val  string `json:"val"`
}

type ServerMap []struct {
	DomainTypeId int               `json:"domain_type_id"`
	Val          string            `json:"val"`
	Rtn          ReqRtn            `json:"rtn"`
	ClientID     int               `json:"client_id"`
	URLPath      string            `json:"url_path"`
	Qs           map[string]string `json:"qs,omitempty"`
	Grouping     string            `json:"grouping"`
	DomainName   string            `json:"domain_name"`
}

type S3Config struct {
	Bucket    string
	Prefix    string
	ServerMap string
	Region    string
}

type CloudWatchEvent struct {
	Version    string          `json:"version"`
	ID         string          `json:"id"`
	DetailType string          `json:"detail-type"`
	Source     string          `json:"source"`
	AccountID  string          `json:"account"`
	Time       time.Time       `json:"time"`
	Region     string          `json:"region"`
	Resources  []string        `json:"resources"`
	Detail     json.RawMessage `json:"detail"`
}