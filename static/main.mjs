import Hls from 'hls'
import {Time, timeToHuman} from 'helpers'

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
})

const onFirstPlay =() => {
	hls.loadSource('http://192.168.23.1:8888/camera/index.m3u8')
	hls.attachMedia(video)
	video.removeEventListener('play', onFirstPlay)
}
video.addEventListener('play', onFirstPlay)

let ctsLatencyInput = document.getElementById('cts-latency')
/**
 * Camera-to-server latency.
 */
let ctsLatencyMs = Number(ctsLatencyInput.value)
ctsLatencyInput.addEventListener('change', e => {
	ctsLatencyMs = Number(ctsLatencyInput.value)
})

/**
 * Latency from the browser -> media server -> camera.
 */
let latencyFromAirMs = 0
let latencyDisplayElement = document.getElementById('latency')

hls.on(Hls.Events.FRAG_CHANGED, function (event, data) {
	if (data.frag.programDateTime !== undefined) {
		latencyFromAirMs = (Date.now() - data.frag.programDateTime + ctsLatencyMs)
		latencyDisplayElement.innerText = '-' + timeToHuman(
			latencyFromAirMs,
			Time.Millis,
			Time.Seconds,
			1,
		)
	}
});

document.getElementById('skip-to-live').addEventListener('click', () => {
	video.currentTime = video.duration - 0.5
})

const savedLatencies = document.getElementById('saved-latencies')
const gotoElement = document.getElementById('goto')
let latencyModeAdd = true
function setSaveLatencyMode(isModeAdd) {
	if (isModeAdd) {
		latencyModeAdd = true
		gotoElement.classList.remove('mode-remove')
	} else {
		latencyModeAdd = false
		gotoElement.classList.add('mode-remove')
	}
}

document.getElementById('save-latency').addEventListener('click', () => {
	setSaveLatencyMode(true)
	const newButton = document.createElement('button')
	const desiredLatencyS = Math.floor(latencyFromAirMs / 1000)
	newButton.innerText = timeToHuman(
		desiredLatencyS,
		Time.Seconds,
		Time.Seconds,
		0,
	)
	newButton.addEventListener('click', () => {
		if (!latencyModeAdd) {
			newButton.remove()
			return;
		}
		video.currentTime = video.duration - desiredLatencyS + ctsLatencyMs / 1000 + 0.5;
	})
	savedLatencies.append(newButton)
})
document.getElementById('remove-latency').addEventListener('click', () => {
	setSaveLatencyMode(!latencyModeAdd)
})

const playbackSpeedInput = document.getElementById('playback-speed-input')
const playbackSpeedOutput = document.getElementById('playback-speed-output')
playbackSpeedInput.addEventListener('input', (event) => {
	updatePlaybackRate(Number(event.target.value))
})
document.getElementById('playback-speed-reset').addEventListener('click', () => {
	resetPlaybackRate()
})
resetPlaybackRate()

/**
 * Sets the playback speed. Larger numbers mean faster. Negative numbers cause slow motion.
 * Zero is no speedup or slowdown.
 * @param {number} input
 */
function updatePlaybackRate(input) {
	if (!Number.isFinite(input)) {
		return;
	}

	let rate = 1
	if (input > 0) {
		rate = input + 1
	} else if (input < 0) {
		rate = input * 0.2 + 1
		rate = Math.max(rate, 0.01)
	}

	playbackSpeedOutput.innerText = '×' + rate.toFixed(2);
	video.playbackRate = rate;
}

function resetPlaybackRate() {
	playbackSpeedInput.value = '0'
	updatePlaybackRate(0)
}

/**
 * Value in seconds that determines that when within this amount of seconds from the
 * camera-to-server latency, the playback speed will be reset.
 */
const resetPlaybackCaughtUpMargin = 2
setInterval(() => {
	if (video.playbackRate > 1
		&& video.duration - video.currentTime < (ctsLatencyMs / 1000) + resetPlaybackCaughtUpMargin) {
		// We caught up to live
		resetPlaybackRate()
	}
}, 1000)

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

