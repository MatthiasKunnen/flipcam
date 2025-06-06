import Hls from 'hls'
import {Time, timeToHuman} from 'helpers'

const videoContainer = document.getElementById('video-container')
const video = document.getElementById('video')
const hls = new Hls({
	// Seconds that frames will be kept in memory. Keeping too many frames is suspected to cause
	// out-of-memory killing on Firefox. The segments are never far away so we don't need to keep
	// many.
	backBufferLength: 3 * 60,
	manifestLoadPolicy: {
		default: {
			maxTimeToFirstByteMs: Infinity,
			maxLoadTimeMs: 20000,
			timeoutRetry: {
				maxNumRetry: 2,
				retryDelayMs: 0,
				maxRetryDelayMs: 0,
			},
			errorRetry: {
				maxNumRetry: 5,
				retryDelayMs: 500,
				maxRetryDelayMs: 2000,
				shouldRetry: (retryConfig, retryCount) => {
					// A 404 error can occur when restarting the muxer
					return retryCount < retryConfig.maxNumRetry;
				},
			},
		},
	},
})

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

const playListUrl = document.getElementById('playlist-url')
const playButton = document.getElementById('playback-start')
playButton.addEventListener('click', () => {
	playButton.style.display = 'none'
	hls.loadSource(playListUrl.value)
	hls.attachMedia(video)
	video.play()
})
playListUrl.addEventListener('change', () => {
	console.log('change', playListUrl.value)
	hls.loadSource(playListUrl.value)
})

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

document.getElementById('fullscreen-toggle').addEventListener('click', () => {
	if (document.fullscreenElement == null) {
		document.documentElement.requestFullscreen({
			navigationUI: 'hide',
		})
	} else {
		document.exitFullscreen();
	}
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

// Expandable and collapsible controls
const controls = document.getElementById('controls')
const main = document.getElementsByTagName('main')[0]
const mainControls = document.getElementById('main-controls')
let canBeFullyVisible = null
let performUpdateWhenNextCollapsed = false
function updateControls() {
	if (performUpdateWhenNextCollapsed) {
		return
	}
	if (videoContainer.contains(mainControls)) {
		performUpdateWhenNextCollapsed = true
		return
	}

	const canBeFullyVisibleNew = isElementInInitialViewport(controls)
	if (canBeFullyVisible === canBeFullyVisibleNew) {
		return
	}
	canBeFullyVisible = canBeFullyVisibleNew

	if (canBeFullyVisible) {
		mainControls.classList.add('no-expand')
	} else {
		mainControls.classList.remove('no-expand')
	}
}

function expandControls() {
	if (videoContainer.contains(controls)) {
		// Do nothing, fully expanded
	} else if (videoContainer.contains(mainControls)) {
		// Go to full expansion
		videoContainer.appendChild(controls)
		controls.prepend(mainControls)
		mainControls.classList.add('fully-expanded')
	} else {
		videoContainer.appendChild(mainControls)
		mainControls.classList.remove('fully-collapsed')
	}
}
function collapseControls() {
	if (videoContainer.contains(controls)) {
		mainControls.classList.remove('fully-expanded')
		main.appendChild(controls)
		videoContainer.appendChild(mainControls)
	} else if (videoContainer.contains(mainControls)) {
		controls.prepend(mainControls)
		mainControls.classList.add('fully-collapsed')

		if (performUpdateWhenNextCollapsed) {
			updateControls()
		}
	}
}

document.getElementById('expand-now').addEventListener('click', () => {
	expandControls()
})
document.getElementById('collapse-now').addEventListener('click', () => {
	collapseControls()
})
window.expand = expandControls
window.collapse = collapseControls

/**
 * @param {HTMLElement} elem
 * @returns boolean
 */
function isElementInInitialViewport(elem) {
	const rect = elem.getBoundingClientRect();
	const scrollTop = window.scrollY;
	const elemTop = rect.top + scrollTop;
	const elemBottom = elemTop + rect.height;
	const viewportHeight = window.innerHeight;
	console.log(elemBottom, viewportHeight)
	return elemTop >= 0 && elemBottom <= viewportHeight;
}

updateControls()
window.addEventListener('resize', () => {
	updateControls()
})

let restarting = false
document.getElementById('restart-muxer').addEventListener('click', () => {
	if (restarting) {
		return
	}
	restarting = true
	const isPlaying = !video.paused && !video.ended;
	(async () => {
	    try {
			const response = await window.fetch('/restart-muxer', {
				method: 'POST',
			})
			if (response.status === 200) {
				const newPath = await response.text()
				/** @var URL */
				let currentUrl
				try {
					currentUrl = new URL(playListUrl.value)
				} catch {
					currentUrl = new URL(document.location.href)
				}
				currentUrl.pathname = newPath
				playListUrl.value = currentUrl.toString()
				hls.loadSource(playListUrl.value)
				if (isPlaying) {
					video.play()
				}
			}
		} catch (error) {
			console.error('Failed to restart muxer: ', error)
		} finally {
			restarting = false
		}
	})().catch(console.error);
})

/** @var {WakeLockSentinel | null} */
let wakeLockSentinel = null;
function updateWakePrevention() {
	if (video.paused) {
		if (wakeLockSentinel === null) {
			// Do nothing
		} else {
			wakeLockSentinel.release().catch(err => console.error('error while releasing sentinel', err))
			wakeLockSentinel = null
		}
	} else {
		if (wakeLockSentinel == null || wakeLockSentinel.released) {
			(async () => {
				wakeLockSentinel = await navigator.wakeLock.request("screen")
				wakeLockSentinel.addEventListener('release', () => {
					updateWakePrevention();
				})
			})().catch(err => console.error('error requesting wakelock', err));
		} else {
			// Do nothing
		}
	}
}
video.addEventListener('play', () => {
	updateWakePrevention()
})
video.addEventListener('pause', () => {
	updateWakePrevention()
})
window.addEventListener('visibilityState', () => {
	updateWakePrevention()
})
