/**
 * Camera availability check.
 *
 * Browsers only expose `navigator.mediaDevices` in a secure context: HTTPS, or
 * a localhost origin. Over plain HTTP on any other host the property is not
 * merely blocked, it is absent, so the first scanner call fails with
 * "Cannot read properties of undefined (reading 'getUserMedia')" — a message
 * that tells the operator nothing about the actual problem.
 *
 * Checking first turns that into an instruction.
 */
export interface CameraSupport {
  supported: boolean
  reason?: 'insecure-context' | 'unsupported-browser'
  message: string
}

export function checkCameraSupport(): CameraSupport {
  const hasMediaDevices =
    typeof navigator !== 'undefined' &&
    typeof navigator.mediaDevices?.getUserMedia === 'function'

  if (hasMediaDevices) {
    return { supported: true, message: '' }
  }

  const insecure = typeof window !== 'undefined' && window.isSecureContext === false
  if (insecure) {
    const origin = typeof window !== 'undefined' ? window.location.origin : 'this address'
    return {
      supported: false,
      reason: 'insecure-context',
      message:
        `The camera is unavailable because ${origin} is not a secure context. ` +
        'Browsers only allow camera access over HTTPS, or on localhost. ' +
        'Serve the app behind HTTPS, or reach it through a localhost tunnel. ' +
        'Manual entry and USB scanners still work here.',
    }
  }

  return {
    supported: false,
    reason: 'unsupported-browser',
    message:
      'This browser does not provide a camera API. ' +
      'Use manual entry or a USB keyboard-wedge scanner instead.',
  }
}
