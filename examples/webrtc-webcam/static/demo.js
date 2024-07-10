let localSessionDescription;
let remoteSessionDescription;
let pc;

async function initWebrtc() {
  pc = new RTCPeerConnection({
    iceServers: [
      {
        urls: "stun:stun.l.google.com:19302",
      },
    ],
  });
  let log = (msg) => {
    document.getElementById("div").innerHTML += msg + "<br>";
  };

  pc.ontrack = function (event) {
    var el = document.createElement(event.track.kind);
    el.srcObject = event.streams[0];
    el.autoplay = true;
    el.controls = true;

    document.getElementById("remoteVideos").appendChild(el);
  };

  pc.oniceconnectionstatechange = (e) => log(pc.iceConnectionState);
  pc.onicecandidate = (event) => {
    if (event.candidate === null) {
      localSessionDescription = btoa(JSON.stringify(pc.localDescription));
    }
  };

  // Offer to receive 1 audio, and 2 video tracks
  pc.addTransceiver("audio", { direction: "sendrecv" });
  pc.addTransceiver("video", { direction: "sendrecv" });
  let d = await pc.createOffer();
  await pc.setLocalDescription(d);
}


initWebrtc();
window.startSession = async () => {
  const resp = await fetch("/start", {
    method: "POST",
    body: JSON.stringify({
      offer: localSessionDescription,
    }),
    headers: {
      "Content-type": "application/json; charset=UTF-8",
    },
  });
  const remoteOffer = await resp.json();
  console.log(remoteOffer);
  remoteSessionDescription = remoteOffer.offer;
  let sd = remoteSessionDescription;
  if (sd === "") {
    return alert("Session Description must not be empty");
  }

  try {
    pc.setRemoteDescription(JSON.parse(atob(sd)));
  } catch (e) {
    alert(e);
  }
};

window.stopSession = async () => {
  const resp = await fetch("/stop", {
    method: "POST",
    headers: {
      "Content-type": "application/json; charset=UTF-8",
    },
  });
  pc.close();
  pc = null;
  let elem = document.getElementById("remoteVideos")
  elem.innerHTML = '';
  initWebrtc();
  console.log("stop");
};
