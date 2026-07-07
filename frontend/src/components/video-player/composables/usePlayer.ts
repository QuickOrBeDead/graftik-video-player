import { computed, reactive, readonly, ref } from 'vue'
import { type PlayerState, RepeatMode, type StreamURLResult, type AppPreferences } from '../types'
import { formatTime } from '../utils'
import { logger } from '@renderer/utils/logger'

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
    streamId: '',
    shouldAutoplay: false
})

export function usePlayer() {
    let lastVolumeLevel = state.volumeLevel
    const repeatModes = [RepeatMode.Off, RepeatMode.All, RepeatMode.One]

    const toggleShuffle = () => {
      state.shuffle = !state.shuffle
      logger.debug('toggleShuffle', state.shuffle)
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
      logger.debug('toggleRepeatMode', state.repeat)
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
      logger.debug('setVolume', state.volumeLevel)
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
      logger.debug('setPlaybackRate', state.playbackRate)
    }

    const timestampLabel = computed(() => {
      return `${formatTime(state.currentTime) } / ${formatTime(state.duration)}`
    })

    const progressBarHoverTime = computed(() => {
      return formatTime(state.progressBarHoverTime)
    })

    const progressPercent = computed(() => {
      return calculatePercent(state.currentTime, state.duration)
    })

    const calculatePercent = (currentTime: number, duration: number): number => {
      if (duration <= 0) {
        return 0
      }

      return (currentTime / duration) * 100
    }

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
      logger.debug('skipTime', seconds, 'newTime', newTime)
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
      logger.debug('seek', percent, 'newTime', newTime)
    }

    const handleProgressBarHover = (percent: number) => {
      state.showProgressBarHoverPreview = true
      state.progressBarHoverTime = getTimeByPercent(percent)
    }

    const hideProgressBarHoverPreview = () => {
      state.showProgressBarHoverPreview = false
    }

    const toVideoUrl = (filePath: string): string => {
      const url = `http://127.0.0.1:${videoPort}/api/video?path=` + encodeURIComponent(filePath)
      logger.debug('toVideoUrl', url)
      return url
    }

    const getStreamUrl = async (playlistItemId: string): Promise<StreamURLResult | null> => {
      logger.debug('getStreamUrl', playlistItemId)
      try {
        const result = await window.go.main.App.GetStreamURL(playlistItemId) as StreamURLResult | null
        logger.debug('getStreamUrl result', result)
        return result
      } catch (e) {
        logger.error('GetStreamURL error:', e)
        return null
      }
    }

    const stopHlsStream = async (streamId: string) => {
      logger.debug('stopHlsStream', streamId)
      try {
        await window.go.main.App.StopHLSStream(streamId)
      } catch (e) {
        logger.error('StopHLSStream error:', e)
      }
    }

    const playVideo = async (videoSrc: string, currentTime: number, playlistItemId?: string) => {
      logger.debug('playVideo', { videoSrc, currentTime, playlistItemId })
      state.playlistItemId = playlistItemId
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
      logger.debug('playVideo: video source set', url)
    }

    const applyPreferences = (preferences: AppPreferences) => {
      logger.debug('applyPreferences', preferences)
      state.shuffle = preferences.shuffle
      state.repeat = preferences.repeatMode as RepeatMode
      state.volumeLevel = preferences.volumeLevel
      state.playbackRate = preferences.playbackRate
      state.sidebarVisible = preferences.sidebarVisible
      state.sidebarWidth = preferences.sidebarWidth
      state.shouldAutoplay = preferences.isPlaying
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
      setSidebarWidth,
      calculatePercent
    }
}
