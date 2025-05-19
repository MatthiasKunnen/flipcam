package flipcamlib

import (
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

func (f *FlipCam) GenerateCaddyConfig() OrderedObject[interface{}] {
	goProApi := "api.gopro.com"
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
			{
				"match", []OrderedObject[interface{}]{
					{
						{"path", []string{f.HlsUrlPathPrefix + "/*.m3u8"}},
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
			// Camera path is HLS, powered by ffmpeg, which writes the files to disk.
			{
				"match", []OrderedObject[interface{}]{
					{
						{"path", []string{f.HlsUrlPathPrefix + "/*"}},
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
						{"handler", "headers"},
						{
							"response", OrderedObject[interface{}]{
								{
									"add", map[string][]string{
										"access-control-allow-origin": {"http://localhost" + f.UiPort},
									},
								},
							},
						},
					},
					{
						{"handler", "rewrite"},
						{"strip_path_prefix", "/camera"},
					},
					{
						{"handler", "file_server"},
						{"index_names", []string{}},
						{"root", f.HlsOutputDir},
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
									{"dial", "localhost" + f.UiPort},
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

	return config
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
