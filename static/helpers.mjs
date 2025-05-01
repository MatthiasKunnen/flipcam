const units = [
	['h', 3600000],
	['m', 60000],
	['s', 1000],
	['ms', 1]
];

export const Time = Object.freeze({
	Hours: 0,
	Minutes: 1,
	Seconds: 2,
	Millis: 3,
})

/**
 * @typedef {0 | 1 | 2 | 3} TimeUnit
 */

/**
 * @param {number} time
 * @param {TimeUnit} inputUnit
 * @param {TimeUnit} smallestUnit
 * @param {number} fractionDigits
 * @returns string
 */
export function timeToHuman(
	time,
	inputUnit,
	smallestUnit,
	fractionDigits,
) {
	let ms = time * units[inputUnit][1]
	let output = []
	let forceAdd = false;

	for (let i = 0; i < smallestUnit; i++) {
		const result = Math.floor(ms / units[i][1])
		if (forceAdd || result > 0) {
			forceAdd = true;
			ms -= result * units[i][1]
			output.push(`${result}${units[i][0]}`)
		}
	}

	output.push((ms / units[smallestUnit][1])
		.toFixed(fractionDigits) + units[smallestUnit][0])

	return output.join(' ')
}


