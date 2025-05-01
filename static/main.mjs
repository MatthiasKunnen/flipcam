import Hls from 'hls'

const video = document.getElementById('video')
const hls = new Hls()
window.flipcam = {
	hls,
	video,
};

// MEDIA_ATTACHED event is fired by hls object once MediaSource is ready
hls.on(Hls.Events.MEDIA_ATTACHED, function () {
	console.log('video and hls.js are bound together')
})
hls.on(Hls.Events.MANIFEST_PARSED, function (event, data) {
	console.log(`manifest loaded, found ${data.levels.length} quality level`)
	video.play()
})
hls.loadSource('http://192.168.23.1:8888/camera/index.m3u8')
hls.attachMedia(video)

let ctsLatencyInput = document.getElementById('cts-latency')
let cameraToServerLatency = Number(ctsLatencyInput.value)
ctsLatencyInput.addEventListener('change', e => {
	cameraToServerLatency = Number(ctsLatencyInput.value)
})

let latencyFromAirMs = 0
let latencyDisplayElement = document.getElementById('latency')

hls.on(Hls.Events.FRAG_CHANGED, function (event, data) {
	if (data.frag.programDateTime !== undefined) {
		latencyFromAirMs = (Date.now() - data.frag.programDateTime + cameraToServerLatency)
		latencyDisplayElement.innerText = `${(latencyFromAirMs / 1000).toFixed(1)} s`
	}
});

document.getElementById('skip-to-live').addEventListener('click', () => {
	video.currentTime = video.duration - 0.5
})

document.body.addEventListener('click', (event) => {
	const target = event.target?.closest('button');
	if (target == null) {
		return;
	}

	target.animate(
		[
			{backgroundColor: 'var(--button-click-color)'},
			{}, // Animate back to the original background color
		],
		{
			duration: 300,
			easing: 'ease-in-out',
		},
	);
});

const targetLatency = 20;

document.getElementById('back-to-delay').addEventListener('click', () => {
	const targetTime = hls.liveSyncPosition - targetLatency;

	// Ensure the target time is not negative
	if (targetTime >= 0) {
		// Set the video's current time to the target time
		video.currentTime = targetTime;
		console.log('Button clicked: Seeking video to', targetTime, 'seconds behind live.');
	} else {
		console.warn('Button clicked: Calculated target time is negative. Seeking to the beginning instead.');
		video.currentTime = 0; // Or handle as appropriate
	}
})
