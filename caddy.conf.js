// Generates caddy.generated.json
// Usage: node caddy.conf.js > caddy.generated.json

const goproApi = 'api.gopro.com';
/** Used to pass network connection check. */
const goproRoutes = [
	{
		match: [
			{
				path: ['/v1/hello.html'],
			},
		],
		handle: [
			{
				handler: 'static_response',
				body: 'Success',
				status_code: 200,
			},
		],
	},
];

const flipcamRoutes = [
	{ // Camera path is HLS, powered by ffmpeg, which writes the files to disk.
		match: [
			{
				path: [
					'/camera/*',
				],
			},
		],
		handle: [
			{
				handler: 'rewrite',
				strip_path_prefix: '/camera',
			},
			{
				handler: 'file_server',
				index_names: [],
				root: '/mnt/perm_sdcard/hls_out',
			},
		],
		terminal: true,
	},
	{ // UI web server
		handle: [
			{
				handler: 'reverse_proxy',
				upstreams: [
					{
						dial: 'localhost:3000',
					},
				],
				transport: {
					protocol: 'http',
				},
			},
		],
		terminal: true,
	},
];

const server = {
	listen: [
		':80',
		':443',
	],
	protocols: [
		'h1',
		'h2',
		// HTTP3 could be considered but note that it will need performance tweaking
		// before being considered viable.
	],
	routes: [
		{
			match: [
				{
					host: [
						goproApi,
					],
				},
			],
			handle: [
				{
					handler: 'subroute',
					routes: goproRoutes,
				},
			],
		},
		{
			match: [
				{
					host: [
						'flipcam.sd4u.be',
					],
				},
			],
			handle: [
				{
					handler: 'subroute',
					routes: flipcamRoutes,
				},
			],
		},
	],
	automatic_https: {
		disable_redirects: true, // The gopro connectivity check may not be redirected to HTTPS
		skip_certificates: [
			goproApi,
		],
	},
};

const config = {
	admin: {
		disabled: true,
	},
	apps: {
		http: {
			servers: {
				srv0: server,
			},
		},
		tls: {
			automation: {
				policies: [
					{
						issuers: [
							{
								module: 'acme',
								challenges: {
									dns: {
										provider: {
											name: 'manual_dns',
											wait_in_mins: '1',
										},
									},
								},
							},
						],
					},
				],
			},
		},
	},
};

console.log(JSON.stringify(config, undefined, '\t'));
