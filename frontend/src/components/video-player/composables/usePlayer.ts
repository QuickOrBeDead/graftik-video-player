import { computed, reactive, readonly, ref } from 'vue'
import { type PlayerState, RepeatMode, type StreamURLResult, type AppPreferences } from '../types'
import { formatTime } from '../utils'

let videoPort: number | null = null

export function setVideoPort(port: number) {
  videoPort = port
}

const state = reactive<PlayerState>({
    isPlaying: false,
    isMuted: false,
    isFullScreen: false,
    volumeLevel: 1.0,
    playbackRate: 1.0,
    shuffle: false,
    repeat: RepeatMode.Off,
    controlsVisible: true,
    currentTime: 0,
    seekTime: 0,
    duration: 0,
    pictureInPicture: false,
    fullScreen: false,
    sidebarVisible: true,
    showProgressBarHoverPreview: false,
    isSidebarResizing: false,
    sidebarWidth: 300,
    streamId: ''
})

export function usePlayer() {
    let lastVolumeLevel = state.volumeLevel
    const repeatModes = [RepeatMode.Off, RepeatMode.All, RepeatMode.One]
    const shouldAutoplay = ref(false)

    const toggleShuffle = () => {
      state.shuffle = !state.shuffle
    }

    const toggleControlsVisible = () => {
      state.controlsVisible = !state.controlsVisible
    }

    const togglePlay = () => {
      state.isPlaying = !state.isPlaying
    }

    const play = () => {
      state.isPlaying = true
    }

    const pause =() => {
      state.isPlaying = false
    }

    const toggleRepeatMode = () => {
      const currentIndex = repeatModes.indexOf(state.repeat)
      state.repeat = repeatModes[(currentIndex + 1) % repeatModes.length]
    }

    const toggleMute = () => {
      if (state.isMuted) {
        state.volumeLevel = lastVolumeLevel
        state.isMuted = false
      } else {
        lastVolumeLevel = state.volumeLevel
        state.isMuted = true
      }
    }

    const setVolume = (level: number) => {
      lastVolumeLevel = state.volumeLevel

      state.volumeLevel = Math.max(0, Math.min(1, level))
      state.isMuted = state.volumeLevel === 0
    }

    const volumeIcon = computed(() => {
      if (state.isMuted || state.volumeLevel == 0) {
        return 'bi bi-volume-mute-fill fs-4 text-danger'
      }

      if (state.volumeLevel < 0.5) {
        return 'bi bi-volume-down-fill fs-4'
      }

      return 'bi bi-volume-up-fill fs-4'
    })

    const setPlaybackRate = (level: number) => {
      state.playbackRate = Math.max(0.5, Math.min(2.0, level))
    }

    const timestampLabel = computed(() => {
      return `${formatTime(state.currentTime) } / ${formatTime(state.duration)}`
    })

    const progressBarHoverTime = computed(() => {
      return formatTime(state.progressBarHoverTime)
    })

    const progressPercent = computed(() => {
      if (state.duration <= 0) {
        return 0
      }

      return (state.currentTime / state.duration) * 100
    })

    const skipTime = (seconds: number) => {
      let newTime = state.currentTime + seconds
      if (newTime < 0) {
        newTime = 0
      }

      if (newTime > state.duration) {
        newTime = state.duration
      }

      state.currentTime = newTime
      state.seekTime = newTime
    }

    const updateTime = (currentTime: number, duration :number) => {
      state.currentTime = currentTime
      state.duration = duration
    }

    const togglePictureInPicture = () => {
      state.pictureInPicture = !state.pictureInPicture
    }

    const toggleFullScreen = () => {
      state.fullScreen = !state.fullScreen
    }

    const toggleSidebarVisible = () => {
      state.sidebarVisible = !state.sidebarVisible
    }

    const getTimeByPercent = (percent: number) => {
      if (percent <= 0) {
        return 0
      }

      if (percent >= 1) {
        return state.duration
      }

      return percent * state.duration
    }

    const seek = (percent: number) => {
      let newTime: number
      if (percent <= 0) {
        newTime = 0
      } else if (percent >= 1) {
        newTime = state.duration
      } else {
        newTime = getTimeByPercent(percent)
      }

      state.currentTime = newTime
      state.seekTime = newTime
    }

    const handleProgressBarHover = (percent: number) => {
      state.showProgressBarHoverPreview = true
      state.progressBarHoverTime = getTimeByPercent(percent)
    }

    const hideProgressBarHoverPreview = () => {
      state.showProgressBarHoverPreview = false
    }

    const toVideoUrl = (filePath: string): string => {
      return `http://127.0.0.1:${videoPort}/api/video?path=` + encodeURIComponent(filePath)
    }

    const getStreamUrl = async (playlistItemId: string): Promise<StreamURLResult | null> => {
      try {
        const result = await window.go.main.App.GetStreamURL(playlistItemId) as StreamURLResult | null
        return result
      } catch (e) {
        console.error('GetStreamURL error:', e)
        return null
      }
    }

    const stopHlsStream = async (streamId: string) => {
      try {
        await window.go.main.App.StopHLSStream(streamId)
      } catch (e) {
        console.error('StopHLSStream error:', e)
      }
    }

    const playVideo = async (videoSrc: string, currentTime: number, playlistItemId?: string) => {
      state.currentTime = currentTime

      if (state.streamId) {
        const oldStreamId = state.streamId
        state.streamId = ''
        stopHlsStream(oldStreamId)
      }

      let url: string
      if (playlistItemId) {
        const result = await getStreamUrl(playlistItemId)
        if (result) {
          url = result.url
          if (result.streamId) {
            state.streamId = result.streamId
          }
        } else {
          url = toVideoUrl(videoSrc)
        }
      } else {
        url = toVideoUrl(videoSrc)
      }

      state.videoSrc = url
      state.isPlaying = true
    }

    const applyPreferences = (prefs: AppPreferences) => {
      state.shuffle = prefs.shuffle
      state.repeat = prefs.repeatMode as RepeatMode
      state.volumeLevel = prefs.volumeLevel
      state.playbackRate = prefs.playbackRate
      state.sidebarVisible = prefs.sidebarVisible
      state.sidebarWidth = prefs.sidebarWidth
      shouldAutoplay.value = prefs.isPlaying
    }

    const setSidebarWidth = (width: number) => {
      state.sidebarWidth = width
    }

    const startSidebarResizing = () => {
      state.isSidebarResizing = true
    }

    const doSidebarResize = (newWidth: number) => {
      if (newWidth > 230 && newWidth < 600) {
        state.sidebarWidth = newWidth
      }
    }

    const stopSidebarResizing = () => {
      state.isSidebarResizing = false
    }

    return {
      playerState: readonly(state),
      shouldAutoplay,
      toggleMute,
      toggleShuffle,
      toggleRepeatMode,
      toggleControlsVisible,
      togglePlay,
      play,
      pause,
      playVideo,
      setVolume,
      setPlaybackRate,
      timestampLabel,
      skipTime,
      volumeIcon,
      updateTime,
      progressPercent,
      togglePictureInPicture,
      toggleSidebarVisible,
      toggleFullScreen,
      seek,
      handleProgressBarHover,
      progressBarHoverTime,
      hideProgressBarHoverPreview,
      startSidebarResizing,
      stopSidebarResizing,
      doSidebarResize,
      toVideoUrl,
      getStreamUrl,
      stopHlsStream,
      applyPreferences,
      setSidebarWidth
    }
}
