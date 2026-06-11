console.log("INDEX.JS LOADED")
const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
const ws = new WebSocket(`${proto}//${window.location.host}/ws/video`)
console.log("ws:",ws)
const peers = {}
const pendingCandidates = {};
let userId;
let localStream;
const localVideo = document.getElementById("local");
console.log(localVideo)
async function startCamera() {
    try {
        console.log("requesting camera");

        localStream =
            await navigator.mediaDevices.getUserMedia({
                video: true,
                audio: true,
            });

        console.log("camera obtained");
        console.log(localStream);

        localVideo.srcObject = localStream;
    } catch (err) {
        console.error("camera error", err);
    }
}
function createPeers(remoteId){
    if (!localStream){
        console.error("localstream not ready");
        return;
    }
    if (peers[remoteId]){return peers[remoteId]}
    const pc = new RTCPeerConnection({
        iceServers: [
            {
                urls: "stun:stun.l.google.com:19302",
            },
        ],
    })

    localStream.getTracks().forEach(track => {
            pc.addTrack(track,localStream)
    });

    pc.onicecandidate = event =>{
        if (!event.candidate) return;
        ws.send(JSON.stringify({
            "type":"candidate",
            "to":remoteId,
            "from":userId,
            "candidate":event.candidate
        }))
    };

    pc.onconnectionstatechange = () => {
    console.log(
        remoteId,
        "connectionState:",
        pc.connectionState
    );
    };

pc.oniceconnectionstatechange = () => {
    console.log(
        remoteId,
        "iceState:",
        pc.iceConnectionState
    );
};

    pc.ontrack = event => {
        console.log(
        "TRACK RECEIVED FROM",
        remoteId
        );
        let video =
            document.getElementById("remote-" + remoteId);

        if (!video) {

            video =
                document.createElement("video");

            video.id = "remote-" + remoteId;
            video.autoplay = true;
            video.playsInline = true;
            video.width = 300;

            document
            .getElementById("videos")
            .appendChild(video);
        }

        video.srcObject = event.streams[0];
    };

    peers[remoteId] = pc;

    return pc;

};

const cameraloading = startCamera();



ws.onmessage = async (event) =>{
    const msg = JSON.parse(event.data);
    console.log(event.data);

    switch(msg.type){
        case "userId":
            userId = msg.userId;
            console.log("userId of current User:",userId);
            break;
        case "existing_user": {
            await cameraloading;

            console.log(
                "existing users",
                msg.users
            );

            for (const remoteId of msg.users) {

                if (remoteId === userId) {
                    continue;
                }

                const pc = createPeers(remoteId);

                const offer =
                    await pc.createOffer();

                await pc.setLocalDescription(
                    offer
                );

                ws.send(JSON.stringify({
                    type: "offer",
                    from: userId,
                    to: remoteId,
                    offer: offer,
                }));
            }

            break;
        }
        case "user_joined":{
            console.log("User Joined",msg.userId)
            break;
        }
        case "offer":{
            console.log("offer received")
            await cameraloading
            console.log("camera ready for offer")
            const pc = createPeers(msg.from);
            await pc.setRemoteDescription(new RTCSessionDescription(msg.offer));
            if (pendingCandidates[msg.from]) {
                for (
                    const candidate
                    of pendingCandidates[msg.from]
                ) {
                    await pc.addIceCandidate(
                        new RTCIceCandidate(candidate)
                    );
                }

                delete pendingCandidates[msg.from];
            }
            const answer = await pc.createAnswer();
            await pc.setLocalDescription(answer);
            ws.send(JSON.stringify({
                "type":"answer",
                "from":userId,
                "to":msg.from,
                "answer":answer,
            }));
            break;
        }
        case "answer":{
            console.log("answer")
            const pc = peers[msg.from];
            if (!pc) {
                console.error(
                    "peer not found for answer",
                    msg.from
                );
                break;
            }
            await pc.setRemoteDescription(new RTCSessionDescription(msg.answer));
            break;
        }
        case "candidate": {
            console.log("candidate")
            const pc = peers[msg.from];

            if (!pc) {

                if (!pendingCandidates[msg.from]) {
                    pendingCandidates[msg.from] = [];
                }

                pendingCandidates[msg.from].push(
                    msg.candidate
                );

                break;
            }

            await pc.addIceCandidate(
                new RTCIceCandidate(msg.candidate)
            );

            break;
        }
        case "user_left":{
            console.log("user left",msg.userId)
            const pc = peers[msg.userId]
            if (pc){
                pc.close()
                delete peers[msg.userId]
            }
            const video =
                document.getElementById(
                    "remote-" + msg.userId
                );

            if (video) {
                video.remove();
            }
            break;
        }
    }
}

function addRemoteVideo(userId, stream) {
    let video =
        document.getElementById(
            `remote-${userId}`
        );

    if (!video) {

        video =
            document.createElement(
                "video"
            );

        video.id =
            `remote-${userId}`;

        video.autoplay = true;
        video.playsInline = true;

        video.style.width = "100%";
        video.style.aspectRatio = "16 / 9";
        video.style.objectFit = "cover";
        video.style.background = "black";
        video.style.borderRadius = "8px";

        document
            .getElementById("videos")
            .appendChild(video);
    }

    video.srcObject = stream;
}

function toggleFullscreen() {
  const wrap = document.getElementById("playerWrap");
  if (!document.fullscreenElement) {
    wrap.requestFullscreen();
  } else {
    document.exitFullscreen();
  }
}
document.addEventListener("fullscreenchange", () => {
  document.getElementById("fsBtn").textContent =
    document.fullscreenElement ? "⊡ Exit" : "⛶ Fullscreen";
});