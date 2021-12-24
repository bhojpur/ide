// Copyright (c) 2018 Bhojpur Consulting Private Limited, India. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	supervisor "github.com/bhojpur/platform/supervisor/api"
	"golang.org/x/xerrors"
	"google.golang.org/grpc"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <port> [<link label>] [<schema>]\n", os.Args[0])
		os.Exit(1)
	}
	port := os.Args[1]

	label := "Open in VS Code Desktop"
	if len(os.Args) > 2 {
		label = os.Args[2]
	}

	schema := "vscode"
	if len(os.Args) > 3 {
		schema = os.Args[3]
	}

	errlog := log.New(os.Stderr, "VS Code Desktop status: ", log.LstdFlags)

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {

		wsInfo, err := GetWSInfo(context.Background())
		if err != nil {
			errlog.Printf("cannot get application info: %v\n", err)
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}

		type Query struct {
			InstanceId  string `json:"instanceId"`
			ApplicationId string `json:"applicationId"`
			PlatformHost  string `json:"platformHost"`
		}
		query := &Query{
			InstanceId:  wsInfo.InstanceId,
			ApplicationId: wsInfo.ApplicationId,
			PlatformHost:  wsInfo.PlatformHost,
		}
		b, err := json.Marshal(query)
		if err != nil {
			errlog.Printf("cannot marshal query: %v\n", err)
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
		queryString := string(b)

		link := url.URL{
			Scheme:   schema,
			Host:     "bhojpur.platform-desktop",
			Path:     wsInfo.CheckoutLocation,
			RawQuery: url.QueryEscape(queryString),
		}

		response := make(map[string]string)
		response["link"] = link.String()
		response["label"] = label
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	fmt.Printf("Starting status proxy for desktop IDE at port %s\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatal(err)
	}
}

func GetWSInfo(ctx context.Context) (*supervisor.ApplicationInfoResponse, error) {
	supervisorAddr := os.Getenv("SUPERVISOR_ADDR")
	if supervisorAddr == "" {
		supervisorAddr = "localhost:22999"
	}
	supervisorConn, err := grpc.Dial(supervisorAddr, grpc.WithInsecure())
	if err != nil {
		return nil, xerrors.Errorf("failed connecting to supervisor: %w", err)
	}
	defer supervisorConn.Close()
	wsinfo, err := supervisor.NewInfoServiceClient(supervisorConn).ApplicationInfo(ctx, &supervisor.ApplicationInfoRequest{})
	if err != nil {
		return nil, xerrors.Errorf("failed getting application info from supervisor: %w", err)
	}
	return wsinfo, nil
}
