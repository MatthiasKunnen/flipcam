package flipcamlib

import (
	"fmt"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"io"
)

var goProApi = "api.gopro.com"

func (f *FlipCam) GenerateCaddyConfig(w io.Writer) error {
	goProRoutes := []OrderedObject[interface{}]{
		{
			{
				"match", []OrderedObject[interface{}]{
					{
						{"path", []string{"/v1/hello.html"}},
					},
				},
			},
			{
				"handle", []OrderedObject[interface{}]{
					{
						{"handler", "static_response"},
						{"body", "Success"},
						{"status_code", 200},
					},
				},
			},
		},
	}

	flipcamRoutes := []OrderedObject[interface{}]{
		{
			// Allow CORS from local origins
			{
				"match", []OrderedObject[interface{}]{
					{
						{"path", []string{f.hlsUrlPathPrefix + "/*.m3u8"}},
					},
				},
			},
			{
				"handle", []OrderedObject[interface{}]{
					{
						{"handler", "headers"},
						{
							"response", OrderedObject[interface{}]{
								{
									"add", map[string][]string{
										"cache-control": {"no-store"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			// Allow cross-origin requests from UI port
			{
				"match", []OrderedObject[interface{}]{
					{
						{"path", []string{f.hlsUrlPathPrefix + "/*"}},
						{
							"header_regexp", OrderedObject[interface{}]{
								{
									"origin", OrderedObject[interface{}]{
										{"name", "cors_local"},
										{"pattern", "^https?:\\/\\/(localhost|127\\.0\\.0\\.1)(:\\d+)?$"},
									},
								},
							},
						},
					},
				},
			},
			{
				"handle", []OrderedObject[interface{}]{
					{
						{"handler", "headers"},
						{
							"response", OrderedObject[interface{}]{
								{
									"set", map[string][]string{
										"access-control-allow-origin": {"{http.request.header.Origin}"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			// Camera path is HLS, powered by ffmpeg, which writes the files to disk.
			{
				"match", []OrderedObject[interface{}]{
					{
						{"path", []string{f.hlsUrlPathPrefix + "/*"}},
					},
				},
			},
			{
				"handle", []OrderedObject[interface{}]{
					{
						{"handler", "encode"},
						{
							"encodings", OrderedObject[interface{}]{
								{
									"gzip", map[string]interface{}{
										"level": 4,
									},
								},
								{
									"zstd", map[string]string{
										"level": "fastest",
									},
								},
							},
						},
						{"prefer", []string{"zstd", "gzip"}},
						{
							"match", OrderedObject[interface{}]{
								{
									"headers", OrderedObject[[]string]{
										{"content-type", []string{"audio/x-mpegurl"}},
									},
								},
							},
						},
						{"minimum_length", 1_000},
					},
					{
						{"handler", "rewrite"},
						{"strip_path_prefix", "/camera"},
					},
					{
						{"handler", "file_server"},
						{"index_names", []string{}},
						{"root", f.hlsOutputDir},
					},
				},
			},
			{"terminal", true},
		},
		{
			// UI web server
			{
				"handle", []OrderedObject[interface{}]{
					{
						{"handler", "reverse_proxy"},
						{
							"upstreams", []OrderedObject[interface{}]{
								{
									{"dial", "localhost" + f.uiPort},
								},
							},
						},
						{
							"transport", OrderedObject[interface{}]{
								{"protocol", "http"},
							},
						},
					},
				},
			},
			{"terminal", true},
		},
	}

	server := OrderedObject[interface{}]{
		{"listen", []string{":80", ":443"}},
		// HTTP3 could be considered but note that it will need performance tweaking
		// before being considered viable.
		{"protocols", []string{"h1", "h2"}},
		{
			"routes", []OrderedObject[interface{}]{
				{
					{
						"match", []OrderedObject[interface{}]{
							{
								{"host", []string{goProApi}},
							},
						},
					},
					{
						"handle", []OrderedObject[interface{}]{
							{
								{"handler", "subroute"},
								{"routes", goProRoutes},
							},
						},
					},
				},
				{
					{
						"match", []OrderedObject[interface{}]{
							{
								{"host", []string{host}},
							},
						},
					},
					{
						"handle", []OrderedObject[interface{}]{
							{
								{"handler", "subroute"},
								{"routes", flipcamRoutes},
							},
						},
					},
				},
			},
		},
		{
			"automatic_https", OrderedObject[interface{}]{
				// The gopro connectivity check may not be redirected to HTTPS
				{"disable_redirects", true},
				{"skip_certificates", []string{goProApi}},
			},
		},
	}

	config := OrderedObject[interface{}]{
		{
			"admin", OrderedObject[interface{}]{
				{"disabled", true},
			},
		},
		{
			"apps", OrderedObject[interface{}]{
				{
					"http", OrderedObject[interface{}]{
						{
							"servers", OrderedObject[interface{}]{
								{"srv0", server},
							},
						},
					},
				},
				{
					"tls", OrderedObject[interface{}]{
						{
							"automation", OrderedObject[interface{}]{
								{
									"policies", []OrderedObject[interface{}]{
										{
											{
												"subjects", []string{host},
											},
											{
												"issuers", []OrderedObject[interface{}]{
													{
														{"module", "acme"},
														{
															"challenges", OrderedObject[interface{}]{
																{
																	"dns", OrderedObject[interface{}]{
																		{
																			"provider", OrderedObject[interface{}]{
																				{"name", "manual_dns"},
																				{"wait_in_mins", "1"},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	err := json.MarshalWrite(w, config, jsontext.WithIndent("\t"))
	if err != nil {
		return err
	}

	_, err = w.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("failed to append EOF newline: %w", err)
	}

	return nil
}

type OrderedObject[V any] []ObjectMember[V]

// ObjectMember is a JSON object member.
type ObjectMember[V any] struct {
	Name  string
	Value V
}

// MarshalJSONTo encodes obj as a JSON object into enc.
func (obj *OrderedObject[V]) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for i := range *obj {
		member := &(*obj)[i]
		if err := json.MarshalEncode(enc, &member.Name); err != nil {
			return err
		}
		if err := json.MarshalEncode(enc, &member.Value); err != nil {
			return err
		}
	}
	if err := enc.WriteToken(jsontext.EndObject); err != nil {
		return err
	}
	return nil
}
