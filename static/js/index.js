console.log("INDEX.JS LOADED");
console.log("room:", ROOM_CODE);

const proto =
    window.location.protocol === "https:"
        ? "wss:"
        : "ws:";

const ws = new WebSocket(
    `${proto}//${window.location.host}/ws/video?code=${ROOM_CODE}`
);

console.log(ws)

const peers = {};
const pendingCandidates = {};

let userId;
let localStream;

const localVideo =
    document.getElementById("local");

async function startCamera() {
    try {
        console.log("requesting camera");

        localStream =
            await navigator.mediaDevices.getUserMedia({
                video: true,
                audio: true,
            });

        localVideo.srcObject =
            localStream;

        console.log(
            "camera obtained"
        );

    } catch (err) {
        console.error(
            "camera error",
            err
        );
    }
}

function createPeer(remoteId) {

    if (peers[remoteId]) {
        return peers[remoteId];
    }

    if (!localStream) {
        console.error(
            "local stream not ready"
        );
        return null;
    }

    const pc =
        new RTCPeerConnection({
            iceServers: [
                {
                    urls:
                        "stun:stun.l.google.com:19302",
                },
            ],
        });

    localStream
        .getTracks()
        .forEach(track => {
            pc.addTrack(
                track,
                localStream
            );
        });

    pc.onicecandidate = event => {

        if (!event.candidate) {
            return;
        }

        ws.send(
            JSON.stringify({
                type: "candidate",
                from: userId,
                to: remoteId,
                candidate:
                    event.candidate,
            })
        );
    };

    pc.onconnectionstatechange =
        () => {
            console.log(
                remoteId,
                "connection:",
                pc.connectionState
            );
        };

    pc.oniceconnectionstatechange =
        () => {
            console.log(
                remoteId,
                "ice:",
                pc.iceConnectionState
            );
        };

    pc.ontrack = event => {

        console.log(
            "TRACK RECEIVED FROM",
            remoteId
        );

        let video =
            document.getElementById(
                "remote-" +
                    remoteId
            );

        if (!video) {

            video =
                document.createElement(
                    "video"
                );

            video.id =
                "remote-" +
                remoteId;

            video.autoplay = true;
            video.playsInline = true;

            video.style.width =
                "100%";

            video.style.maxWidth =
                "450px";

            video.style.aspectRatio =
                "16/9";

            video.style.objectFit =
                "cover";

            video.style.borderRadius =
                "8px";

            document
                .getElementById(
                    "videos"
                )
                .appendChild(
                    video
                );
        }

        video.srcObject =
            event.streams[0];

        video.play().catch(
            err => {
                console.log(
                    "play blocked",
                    err
                );
            }
        );
    };

    peers[remoteId] = pc;

    return pc;
}

const cameraLoading =
    startCamera();

ws.onopen = () => {
    console.log(
        "websocket connected"
    );
};

ws.onerror = err => {
    console.error(
        "websocket error",
        err
    );
};

ws.onclose = () => {
    console.log(
        "websocket disconnected"
    );
};

ws.onmessage = async event => {

    const msg =
        JSON.parse(event.data);

    console.log(
        "WS:",
        msg
    );

    switch (msg.type) {

        case "userId": {

            userId =
                msg.userId;

            console.log(
                "my id:",
                userId
            );

            break;
        }

        case "existing_user": {

            await cameraLoading;

            console.log(
                "existing users:",
                msg.users
            );

            for (
                const remoteId
                of msg.users
            ) {

                if (
                    remoteId ===
                    userId
                ) {
                    continue;
                }

                if (
                    peers[
                        remoteId
                    ]
                ) {
                    continue;
                }

                const pc =
                    createPeer(
                        remoteId
                    );

                const offer =
                    await pc.createOffer();

                await pc.setLocalDescription(
                    offer
                );

                ws.send(
                    JSON.stringify({
                        type:
                            "offer",
                        from:
                            userId,
                        to:
                            remoteId,
                        offer:
                            offer,
                    })
                );
            }

            break;
        }

        case "user_joined": {

            console.log(
                "user joined:",
                msg.userId
            );

            break;
        }

        case "offer": {

            console.log(
                "offer received from",
                msg.from
            );

            await cameraLoading;

            const pc =
                createPeer(
                    msg.from
                );

            await pc.setRemoteDescription(
                new RTCSessionDescription(
                    msg.offer
                )
            );

            if (
                pendingCandidates[
                    msg.from
                ]
            ) {

                for (
                    const candidate of
                    pendingCandidates[
                        msg.from
                    ]
                ) {

                    await pc.addIceCandidate(
                        new RTCIceCandidate(
                            candidate
                        )
                    );
                }

                delete pendingCandidates[
                    msg.from
                ];
            }

            const answer =
                await pc.createAnswer();

            await pc.setLocalDescription(
                answer
            );

            ws.send(
                JSON.stringify({
                    type:
                        "answer",
                    from:
                        userId,
                    to:
                        msg.from,
                    answer:
                        answer,
                })
            );

            break;
        }

        case "answer": {

            console.log(
                "answer from",
                msg.from
            );

            const pc =
                peers[msg.from];

            if (!pc) {
                console.error(
                    "peer not found",
                    msg.from
                );
                break;
            }

            await pc.setRemoteDescription(
                new RTCSessionDescription(
                    msg.answer
                )
            );

            if (
                pendingCandidates[
                    msg.from
                ]
            ) {

                for (
                    const candidate of
                    pendingCandidates[
                        msg.from
                    ]
                ) {

                    await pc.addIceCandidate(
                        new RTCIceCandidate(
                            candidate
                        )
                    );
                }

                delete pendingCandidates[
                    msg.from
                ];
            }

            break;
        }

        case "candidate": {

            const pc =
                peers[msg.from];

            if (!pc) {

                pendingCandidates[
                    msg.from
                ] ??= [];

                pendingCandidates[
                    msg.from
                ].push(
                    msg.candidate
                );

                break;
            }

            if (
                pc.remoteDescription
            ) {

                await pc.addIceCandidate(
                    new RTCIceCandidate(
                        msg.candidate
                    )
                );

            } else {

                pendingCandidates[
                    msg.from
                ] ??= [];

                pendingCandidates[
                    msg.from
                ].push(
                    msg.candidate
                );
            }

            break;
        }

        case "user_left": {

            console.log(
                "user left:",
                msg.userId
            );

            const pc =
                peers[
                    msg.userId
                ];

            if (pc) {

                pc.close();

                delete peers[
                    msg.userId
                ];
            }

            delete pendingCandidates[
                msg.userId
            ];

            const video =
                document.getElementById(
                    "remote-" +
                        msg.userId
                );

            if (video) {
                video.remove();
            }

            break;
        }
    }
};

function toggleFullscreen() {

    const wrap =
        document.getElementById(
            "playerWrap"
        );

    if (
        !document.fullscreenElement
    ) {

        wrap.requestFullscreen();

    } else {

        document.exitFullscreen();
    }
}

document.addEventListener(
    "fullscreenchange",
    () => {

        document.getElementById(
            "fsBtn"
        ).textContent =
            document.fullscreenElement
                ? "⊡ Exit"
                : "⛶ Fullscreen";
    }
);