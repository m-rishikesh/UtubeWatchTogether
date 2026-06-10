console.log("INDEX.JS LOADED")
const ws = new WebSocket("wss://utubewatchtogether.onrender.com/ws/video")
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
    }

    pc.ontrack = event => {

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
        case "user_joined":{
            await cameraloading
            const remoteId = msg.userId;
            console.log("UserJoined",remoteId);
            const pc = createPeers(remoteId);
            if (pendingCandidates[remoteId]) {
                for (
                    const candidate
                    of pendingCandidates[remoteId]
                ) {
                    await pc.addIceCandidate(
                        new RTCIceCandidate(candidate)
                    );
                }

                delete pendingCandidates[remoteId];
            }
                        const offer = await pc.createOffer();
            await pc.setLocalDescription(offer);
            ws.send(JSON.stringify({
                "type":"offer",
                "from":userId,
                "to":remoteId,
                "offer":offer,
            }))
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

