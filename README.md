# FlipCam
FlipCam sets up a video stream with advanced controls for the purpose of improving
techniques of athletes using an adjustable, delayed, video feed of their technique.

Examples include, but are not limited to, evaluating; gymnasts, swimmers (flip turns/dives), and
all manner of sports where an immediate, slow motion, replay of athlete's performances can help them
improve their technique.

The idea is to have a camera pointed at the athlete and a device such as a tablet, playing the
video at a custom delay so that they can see their own performance and improve. 

## Features
- Fully offline.
- A mobile-responsive, website to view the camera feed at. Accessible via Wi-Fi.
- Advanced video controls:
  - Playback speed from slow motion (1/100th of actual speed) to fast-forward (6 times speed).
  - Setting a custom delay, for example, the replay can be set to 5 seconds behind reality. 
  - Shortcuts to set certain delays, this allows quick changes of the delay.
- Efficient stream conversion powered by ffmpeg
- Multiple devices can watch the stream with their own custom delays/video speed.
  This can allow the coach and the athletes to analyse separate parts of a performance.
- Medium minimum latency, between two and four seconds is expected.

## How it works
FlipCam does the following:
1. Sets up a Wi-Fi network named `flipcam` with DHCP.
1. Sets up ffmpeg to ingest RTMP (e.g. from a GoPro livestream), and convert it to HLS.
1. Runs a web server using Caddy that provides the video stream and the controls.

Caddy, dnsmasq, and hostapd, are run as systemd services for security purposes.

## Current state
Currently, FlipCam (the server) is built to work on Linux only.
Clients can view the stream on any device as long as it has Wi-Fi and a web browser.

### HTTPS
Without HTTPS, a performance penalty in playback is expected as HTTP2 and HTTP3 are not used by
browsers.
The options are either:
- Self-signed certificate; downside is the HTTPS errors given by browsers which users will have to
  click through.
- Getting a valid certificate for a domain you control and adding an A record to the machine's IP.

By default, self-signed certificates are generated for the IP address and a manual DNS Lets Encrypt
challenge is initiated if `--hostname` is specified.
See the `flipcam-caddy.service` logs. 
Making HTTPS plug-and-play with valid certificates, while possible, would only be feasible for a
commercial product and require a server.
Users should evaluate their preferred HTTPS approach and make changes accordingly.

Note: self-signed certificates are currently not working on Firefox, see
<https://github.com/caddyserver/caddy/issues/6891>.

### Wi-Fi
`hostapd` is used to set up an access point on the selected wireless interface.
Considerations have been made to make this as plug-and-play as possible but some limitations apply.

- The selected interface cannot be connected to another Wi-Fi network while it is active.
  This means that if you have only one network interface, you will not be able to access the internet.
- `hostapd` cannot be run when `iwd` is active.
  If you have several wireless interfaces and want to keep using the non-selected ones, use
  `wpa_supplicant`.
  If you find a solution for this problem, please make an issue.

If you are using NetworkManager, flipcam will automatically set the selected wireless interface
as unmanaged and undo this change when it closes.

## Requirements
- dnsmasq (DHCP and DNS resolving)
- ffmpeg (video stream processing)
- go (to build the binaries)
- hostapd (Wi-Fi access point)
- polkit (managing the systemd services)
- sudo

## Quick start
### Setup
1. `go build .` Builds the flipcam binary.
1.  ```
    ./flipcam genconf \
        --wireless-interface NAME_HERE \
        --wireless-password PASSWORD_HERE \
        --caddy-binary-path /usr/local/bin/caddy \
        --hostname flipcam.optional.com
    ```
   This generates `ansible/generated_vars.yaml` and `ansible/templates/caddy.json`.
   Read, validate, and customize these files if necessary.
1. Run the Ansible playbook, see [ansible/README.md](./ansible/README.md).
1. Add the user that should run flipcam to the flipcam user group. `usermod -aG flipcam username`.
   Relogging is most likely required.

### Run
1. `flipcam run --wireless-interface NAME_HERE`
1. Go to `https://192.168.23.1` or `https://hostname` if a hostname was set (replace the placeholders).
1. Other devices can connect to the flipcam network and visit these addresses.

## Configuring a GoPro

### GoProLabs
GoPro makes a beta firmware that allows your camera to be controlled via QR codes called
[GoPro Labs](https://gopro.github.io/labs/). This can simplify usage and setup.

QR codes are generated in [goprolabs_qrcodes.html](./goprolabs_qrcodes.html).

1. Scan the _Make GoPro join WiFi_ QR code.  
   Change the password if needed, editing the HTML should be straightforward.  
1. Scan the _Set RTMP URL_ QR code.
1. Start a livestream by scanning one of the _Stream_ QR codes.  
   QR codes are divided in GoPro Hero `< 12` and `>=12`.
   Pick the one appropriate for your GoPro.

   If you have a GoPro >=12, and the stream does not want to initialize, try the <12 QR code.
   This might fix the problem for future streams, see
   <https://github.com/gopro/labs/issues/1398#issuecomment-2882051110>

### Using Quik
1. Go to the GoPro menu.
1. Press the three dots next to your device.
1. Click on _Live Stream_.
1. Select _Other/RTMP_.
1. Select the Wi-Fi network flipcam and enter the credentials.
1. Enter the RTMP address, `http://192.168.23.1/camera`.
1. Click on _Continue_.
1. Click on _Go Live_.

## Troubleshooting
See [TROUBLESHOOTING.md](./TROUBLESHOOTING.md).

## TODO
- [ ] DNS resolving on the device is a problem, fix probably with openresolv. NM overwrites the /etc/resolv.conf file but we need it to first resolve the flipcam and gopro dns records before going to other dns resolvers

## Backlog
- [ ] Use HTTP3? A careful review will have to be made. Higher CPU usage and perhaps increased latency combined with worse playback might occur on the device running flipcam, but it could improve latency and playback of devices on the network.
- [ ] Add option to remove fragments periodically, or when flipcam exits.
- [ ] Generate QR codes in web interface instead of separate HTML file
- [ ] Unblock rfkill if blocked
- [ ] When the loading fails, the busy indicator (spinner) continues but the attempts are stopped at
      Some point without indication, fix this. Either reset the spinner or continue attempts.
- [ ] support iwd. hostapd does not want to start when iwd is running.
      When iwd is stopped, the wireless interface disappears and hostapd does not start.
      Perhaps the best way to do this is to use iwd as the AP, see
      <https://archive.kernel.org/oldwiki/iwd.wiki.kernel.org/ap_mode.html>.
      This might have performance considerations.
- [ ] hostapd is reported as started by systemctl before it is actually running. Might need to wrap
      it in a notify-reload script
