import { afterEach, describe, expect, it, vi } from 'vitest'

import { checkCameraSupport } from './camera'

function setEnvironment(options: {
  secure: boolean
  mediaDevices?: { getUserMedia?: unknown }
  origin?: string
}) {
  vi.stubGlobal('window', {
    isSecureContext: options.secure,
    location: { origin: options.origin ?? 'http://192.168.1.50:6002' },
  })
  vi.stubGlobal('navigator', { mediaDevices: options.mediaDevices })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('checkCameraSupport', () => {
  it('reports support when the browser exposes getUserMedia', () => {
    setEnvironment({ secure: true, mediaDevices: { getUserMedia: () => {} } })

    expect(checkCameraSupport()).toEqual({ supported: true, message: '' })
  })

  // This is the Docker-over-HTTP case: mediaDevices is absent entirely, which
  // is what produces "Cannot read properties of undefined".
  it('explains the secure-context rule and names the origin', () => {
    setEnvironment({ secure: false, mediaDevices: undefined, origin: 'http://10.0.0.7:6002' })

    const result = checkCameraSupport()
    expect(result.supported).toBe(false)
    expect(result.reason).toBe('insecure-context')
    expect(result.message).toContain('http://10.0.0.7:6002')
    expect(result.message).toContain('HTTPS')
    // The operator still has a way to work, and the message has to say so.
    expect(result.message).toMatch(/manual entry/i)
  })

  it('distinguishes a browser without a camera API from an insecure origin', () => {
    setEnvironment({ secure: true, mediaDevices: undefined })

    const result = checkCameraSupport()
    expect(result.supported).toBe(false)
    expect(result.reason).toBe('unsupported-browser')
    expect(result.message).not.toContain('HTTPS')
  })

  it('treats a present mediaDevices without getUserMedia as unsupported', () => {
    setEnvironment({ secure: true, mediaDevices: {} })

    expect(checkCameraSupport().supported).toBe(false)
  })
})
