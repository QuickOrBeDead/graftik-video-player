<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import { usePlayer, setVideoPort } from './composables/usePlayer'
import { usePlaylist } from './composables/usePlaylist'
import { RepeatMode } from './types'

const {
    playerState,
    setVolume,
    setPlaybackRate,
    toggleMute,
    toggleShuffle,
    toggleRepeatMode,
    togglePlay,
    play,
    playVideo,
    pause,
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
    hideProgressBarHoverPreview
} = usePlayer()
const { getNextPlaylistItem, getPreviousPlaylistItem, setPlaylistCurrentItem, clearShuffledDeck } = usePlaylist()

const progressBarHoverPreviewX = ref<number>(0)
const previewVideo = ref<HTMLVideoElement | null>(null)
const previewCanvas = ref<HTMLCanvasElement | null>(null)
const videoPlayerElement = ref<HTMLVideoElement | null>(null)

onMounted(async () => {
    calculateVideoHeight()
    const port = await window.go.main.App.GetVideoServerPort()
    setVideoPort(port)
})

watch(() => playerState.seekTime, (newCurrentTime: number) => {
    const v = videoPlayerElement.value
    if (!v) return

    v.currentTime = newCurrentTime
})

watch(() => playerState.videoSrc, async (newVideoSrc, oldVideoSrc) => {
  const v = videoPlayerElement.value
  if (!v) return

  if (newVideoSrc === oldVideoSrc) {
    return
  }

  v.pause()

  if (newVideoSrc) {
    v.src = newVideoSrc
    v.load()
  }
})

watch(() => playerState.isPlaying, async (newPlaying, oldPlaying) => {
  const v = videoPlayerElement.value
  if (!v) return

  if (newPlaying === oldPlaying) {
    return
  }

  if (newPlaying) {
    if (v.paused) {
      await v.play()
    }
  } else {
    v.pause()
  }
})

watch(() => playerState.pictureInPicture, async (newPictureInPicture: boolean) => {
    if (!newPictureInPicture) {
        return
    }

    if (document.pictureInPictureElement) {
        await document.exitPictureInPicture()
    } else if (videoPlayerElement.value) {
        await videoPlayerElement.value.requestPictureInPicture()
    }
})

watch(() => playerState.fullScreen, () => {
    if (!document.fullscreenElement) {
        const v = videoPlayerElement.value
        if (v && v.requestFullscreen) {
            v.requestFullscreen()
        }
    } else {
        if (document.exitFullscreen) {
            document.exitFullscreen()
        }
    }
})

const onMetadataLoaded = () => {
  const v = videoPlayerElement.value
  if (!v) return

  const { currentTime, playbackRate } = playerState

  const initVideo = async () => {
    v.currentTime = currentTime
    v.playbackRate = playbackRate
    v.removeEventListener('canplay', initVideo)

    if (playerState.isPlaying) {
      try {
        await v.play()
      } catch(e) {
        console.error('v.play()', e)
      }
    } else {
      v.pause()
    }
  }

  v.addEventListener('canplay', initVideo, { once: true })
}

const onVideoError = () => {
  const v = videoPlayerElement.value
  if (!v || !v.error) return

  const errorMap: Record<number, string> = {
    1: "Playback aborted by user.",
    2: "Network error: Check your connection.",
    3: "Video decoding failed: Format not supported.",
    4: "Video source not found (404)."
  }

  const errorMessage = errorMap[v.error.code] || "An unknown video error occurred."

  console.error(`Video Error ${v.error.code}: ${errorMessage}`)

  // Optional: Update your UI state to show a message to the user
  // playerState.errorMessage = errorMessage
  // playerState.isPlaying = false
}

const calculateVideoHeight = () => {
  const height = `${Math.max(document.documentElement.clientHeight, window.innerHeight || 0) - 10}px`
  if (videoPlayerElement.value) {
    videoPlayerElement.value.style.height = height
  }
}

const calculateClientXPercent = function(e: { currentTarget: EventTarget | null, clientX: number }) {
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
    const x = e.clientX - rect.left
    if (rect.width === 0) {
        return {
            x,
            percent: 0
        }
    }

    return {
        x,
        percent: Math.max(0, Math.min(1, x / rect.width))
    }
}

const progressBarMouseMove = async (e: MouseEvent) => {
    const { percent, x } = calculateClientXPercent(e)
    progressBarHoverPreviewX.value = x
    handleProgressBarHover(percent)

    await nextTick()

    if (playerState.progressBarHoverTime && previewVideo.value) {
        const v = previewVideo.value
        v.currentTime = playerState.progressBarHoverTime
        v.onseeked = () => {
            if (previewCanvas.value) {
                const ctx = previewCanvas.value.getContext('2d')
                ctx!.drawImage(v, 0, 0, 180, 101)
            }
            v.onseeked = null
        }
    }
}

const progressBarClick = (e: PointerEvent) => {
    const { percent } = calculateClientXPercent(e)
    seek(percent)
}

const onVideoEnded = () => {
    const nextItem = getNextPlaylistItem(playerState.repeat, playerState.shuffle)
    if (nextItem) {
      const restartTime = playerState.repeat === RepeatMode.One ? 0 : (nextItem.elapsedTime ?? 0)
      playVideo(nextItem.path, restartTime)
      setPlaylistCurrentItem(nextItem.id)
    } else {
      if (playerState.shuffle) {
        clearShuffledDeck()
      }
      pause()
    }
}

const playPreviousVideo = () => {
    const prevItem = getPreviousPlaylistItem(playerState.repeat, playerState.shuffle)
    if (prevItem) {
      playVideo(prevItem.path, 0)
      setPlaylistCurrentItem(prevItem.id)
    }
}

const playNextVideo = () => {
    const nextItem = getNextPlaylistItem(playerState.repeat, playerState.shuffle)
    if (nextItem) {
      playVideo(nextItem.path, 0)
      setPlaylistCurrentItem(nextItem.id)
    }
}
</script>

<template>
    <!-- Hidden video for thumbnail generation -->
    <video id="preview-video" ref="previewVideo" :src="playerState.videoSrc" crossorigin="anonymous" muted></video>
    <div class="video-section rounded-2 overflow-hidden">
        <video
            ref="videoPlayerElement"
            @loadedmetadata="onMetadataLoaded"
            @error="onVideoError"
            @ended="onVideoEnded"
            :volume="playerState.volumeLevel"
            :muted="playerState.isMuted"
            :playbackRate="playerState.playbackRate"
            @timeupdate="(e: Event) => {
              const v = e.target as HTMLVideoElement
              if (v.currentTime > v.duration + 1) {
                return
              }

              if (v.readyState >= 1 && !v.seeking) {
                updateTime(v.currentTime, v.duration)
              }
            }"
            @play="play"
            @pause="pause"
            @click="togglePlay"
            crossorigin="anonymous"
            allowfullscreen
        >
            Your computer does not support the video tag.
        </video>

        <!-- Player Controls -->
        <div class="controls-overlay" v-show="playerState.controlsVisible">
            <div
                class="progress-wrapper"
                @click="progressBarClick"
                @mousemove="progressBarMouseMove"
                @mouseleave="hideProgressBarHoverPreview"
            >
            <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>

            <!-- Separated Thumbnail Preview -->
            <div
                v-if="playerState.showProgressBarHoverPreview"
                class="thumbnail-preview"
                :style="{ left: progressBarHoverPreviewX + 'px' }"
            >
                <canvas ref="previewCanvas" class="thumbnail-canvas" width="180" height="101"></canvas>
            </div>

            <!-- Separated Time Signature Tooltip -->
            <div
                v-if="playerState.showProgressBarHoverPreview"
                class="time-tooltip"
                :style="{ left: progressBarHoverPreviewX + 'px' }"
            >
                {{ progressBarHoverTime }}
            </div>
            </div>

            <div class="d-flex justify-content-between align-items-center">
                <!-- Left Controls -->
                <div class="d-flex align-items-center gap-1">
                    <button class="btn btn-icon border-0 bg-transparent" @click="togglePlay" data-bs-toggle="tooltip" :title="playerState.isPlaying ? 'Pause (Space)' : 'Play (Space)'" id="playBtn">
                        <i :class="playerState.isPlaying ? 'bi bi-pause-fill fs-2' : 'bi bi-play-fill fs-2'"></i>
                    </button>

                    <button class="btn btn-icon border-0 bg-transparent" @click="skipTime(-10)" data-bs-toggle="tooltip" title="Backward 10s (Left Arrow)">
                        <i class="bi bi-arrow-counterclockwise fs-4"></i>
                    </button>

                    <button class="btn btn-icon border-0 bg-transparent" @click="skipTime(10)" data-bs-toggle="tooltip" title="Forward 10s (Right Arrow)">
                        <i class="bi bi-arrow-clockwise fs-4"></i>
                    </button>

                    <button class="btn btn-icon border-0 bg-transparent" @click="playPreviousVideo" data-bs-toggle="tooltip" title="Previous (n)">
                        <i class="bi bi-skip-start-fill fs-4"></i>
                    </button>

                    <button class="btn btn-icon border-0 bg-transparent" @click="playNextVideo" data-bs-toggle="tooltip" title="Next (n)">
                        <i class="bi bi-skip-end-fill fs-4"></i>
                    </button>

                    <div class="volume-control ms-2">
                    <button class="btn btn-icon border-0 bg-transparent" @click="toggleMute" data-bs-toggle="tooltip" :title="playerState.isMuted ? 'Unmute (m)' : 'Mute (m)'">
                        <i :class="volumeIcon"></i>
                    </button>
                    <input
                        type="range"
                        class="volume-slider"
                        min="0"
                        max="1"
                        step="0.05"
                        :value="playerState.volumeLevel"
                        @input="(e: Event) => setVolume(parseFloat((e.target as HTMLInputElement).value))"
                    >
                    </div>

                    <span class="small font-monospace ms-3 text-secondary">{{ timestampLabel }}</span>
                </div>

                <!-- Right Controls (Updated Order) -->
                <div class="d-flex align-items-center gap-3">
                    <!-- Shuffle Button -->
                    <button class="btn btn-icon border-0 bg-transparent" :class="{ active: playerState.shuffle }" @click="toggleShuffle" data-bs-toggle="tooltip" title="Shuffle (s)">
                        <i class="bi bi-shuffle fs-5"></i>
                    </button>

                    <!-- Repeat Button -->
                    <button class="btn btn-icon border-0 bg-transparent" :class="{ active: playerState.repeat !== RepeatMode.Off }" @click="toggleRepeatMode" data-bs-toggle="tooltip" title="Repeat Mode (r)">
                        <i class="bi bi-repeat fs-5"></i>
                        <span v-if="playerState.repeat === RepeatMode.One" class="repeat-badge">1</span>
                    </button>

                    <!-- Playback Speed -->
                    <select :value="playerState.playbackRate" @change="(e: Event) => setPlaybackRate(parseFloat((e.target as HTMLSelectElement).value))" class="form-select form-select-sm bg-dark text-white border-secondary" style="width: auto;" data-bs-toggle="tooltip" title="Playback Speed">
                        <option v-for="rate in [0.5, 1, 1.25, 1.5, 2]" :key="rate" :value="rate">{{rate}}x</option>
                    </select>

                    <!-- Picture in Picture -->
                    <button class="btn btn-icon border-0 bg-transparent" @click="togglePictureInPicture" data-bs-toggle="tooltip" title="Picture in Picture (p)">
                        <i class="bi bi-pip fs-5"></i>
                    </button>

                    <!-- Toggle Playlist -->
                    <button class="btn btn-icon border-0 bg-transparent" @click="toggleSidebarVisible" data-bs-toggle="tooltip" title="Toggle Playlist">
                        <i :class="playerState.sidebarVisible ? 'bi bi-layout-sidebar-reverse fs-5' : 'bi bi-layout-sidebar fs-5'"></i>
                    </button>

                    <!-- Full Screen -->
                    <button class="btn btn-icon border-0 bg-transparent" @click="toggleFullScreen" data-bs-toggle="tooltip" title="Full Screen (f)">
                    <i :class="playerState.fullScreen ? 'bi bi-fullscreen-exit fs-5' : 'bi bi-fullscreen fs-5'"></i>
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>
<style lang="css" scoped>
.video-section {
  position: relative;
  flex-grow: 1;
  background: black;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.video-section:fullscreen {
  width: 100vw;
  height: 100vh;
}

video {
  width: 100%;
  max-height: 100%;
  outline: none;
}

.controls-overlay {
  position: absolute;
  bottom: 0;
  width: 100%;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.9));
  padding: 20px 30px;
  z-index: 100;
}

.progress-wrapper {
  height: 6px;
  background: rgba(255, 255, 255, 0.2);
  cursor: pointer;
  border-radius: 3px;
  margin-bottom: 15px;
  position: relative;
  transition: height 0.1s;
}

.progress-wrapper:hover {
  height: 8px;
}

.progress-fill {
  height: 100%;
  background: var(--accent-blue);
  border-radius: 3px;
  position: relative;
}

/* Thumbnail Preview */
.thumbnail-preview {
  position: absolute;
  bottom: 55px; /* Moved up to make room for time signature */
  transform: translateX(-50%);
  background: #000;
  border: 2px solid rgba(255, 255, 255, 0.2);
  border-radius: 8px;
  overflow: hidden;
  pointer-events: none;
  box-shadow: 0 8px 25px rgba(0,0,0,0.8);
  z-index: 120;
}

.thumbnail-canvas {
  width: 180px;
  height: 101px;
  display: block;
  background: #000;
}

/* Distinct Time Signature Tooltip */
.time-tooltip {
  position: absolute;
  bottom: 25px;
  transform: translateX(-50%);
  background: rgba(40, 40, 40, 0.95);
  color: white;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  font-family: monospace;
  pointer-events: none;
  border: 1px solid rgba(255, 255, 255, 0.1);
  z-index: 121;
}

.volume-control {
  display: flex;
  align-items: center;
  gap: 8px;
  transition: width 0.3s;
}

.volume-slider {
  width: 0;
  opacity: 0;
  transition: all 0.2s ease;
  cursor: pointer;
  accent-color: var(--accent-blue);
}

.volume-control:hover .volume-slider {
  width: 80px;
  opacity: 1;
}

.btn-icon {
  color: white;
  padding: 5px;
  border-radius: 50%;
  transition: background 0.2s;
  position: relative;
}

.btn-icon:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--accent-blue);
}

.btn-icon.active {
  color: var(--accent-blue);
}

.repeat-badge {
  position: absolute;
  top: 2px;
  right: 0px;
  font-size: 0.6rem;
  font-weight: bold;
  background: var(--accent-blue);
  color: black;
  border-radius: 50%;
  width: 12px;
  height: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

#preview-video {
  display: none;
}
</style>
