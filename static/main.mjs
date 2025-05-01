import Hls from 'hls'

const video = document.getElementById('video')
const hls = new Hls()
window.hls = hls;

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

hls.on(Hls.Events.FRAG_CHANGED, function (event, data) {
	if (data.frag.programDateTime !== undefined) {
		console.log('Current Program Date Time:', new Date(data.frag.programDateTime).toISOString());
	}
});

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
