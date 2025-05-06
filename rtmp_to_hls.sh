#!/usr/bin/env bash

# Alternative to mediamtx, more performant, simpler.
# Requires a web server to serve the HLS files. This approach prevents the response times
# sky rocketing as stream time increases with mediamtx.

ffmpeg -loglevel warning -listen 1 -i rtmp://0.0.0.0:1935/camera -c:v copy -c:a copy -f hls -hls_list_size 0 -hls_segment_type fmp4 -hls_time 1 -hls_flags program_date_time+split_by_time -hls_playlist_type event -hls_segment_filename /mnt/perm_sdcard/hls_out/_seg%d.mp4 /mnt/perm_sdcard/hls_out/index.m3u8
