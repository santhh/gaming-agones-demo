package com.google.swarm.gaming.matchmaker.controller;

import java.io.IOException;
import java.net.DatagramPacket;
import java.net.DatagramSocket;
import java.net.InetAddress;
import java.util.ArrayList;
import java.util.Iterator;
import java.util.List;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.web.bind.annotation.CrossOrigin;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import com.google.cloud.datastore.Datastore;
import com.google.cloud.datastore.DatastoreOptions;
import com.google.cloud.datastore.Entity;
import com.google.cloud.datastore.Key;
import com.google.cloud.datastore.KeyFactory;
import com.google.cloud.datastore.StringValue;
import com.google.cloud.datastore.Value;
import com.google.common.geometry.S2CellId;
import com.google.common.geometry.S2LatLng;
import com.google.swarm.gaming.matchmaker.GamingMatchmakerServiceApplication.MatchmakerProp;
import com.google.swarm.gaming.matchmaker.client.AllocationServiceClient;
import com.google.swarm.gaming.matchmaker.model.Allocation;
import com.google.swarm.gaming.matchmaker.model.Allocation.Status;
import com.google.swarm.gaming.matchmaker.model.Allocation.Status.Ports;

import lombok.RequiredArgsConstructor;

@RestController
@CrossOrigin
@RequiredArgsConstructor
@RequestMapping(value = "/matchmaker/v1")
public class MatchMakerController {
	static private final Logger LOG = LoggerFactory.getLogger(MatchMakerController.class);

	private final AllocationServiceClient allocationServiceClient;

	private final static Integer MAX_PLAYER_ALLOWED = MatchmakerProp.maxNumberofUsers;

	@GetMapping(value = "/allocate/lat/{lat}/lon/{lon}")
	public Allocation getAllocation(@PathVariable String lat, @PathVariable String lon) throws IOException {

		String cellId;
		Allocation allocation = new Allocation();
		if (lat != null && lon != null) {

			cellId = getCellId(Double.parseDouble(lat), Double.parseDouble(lon));
			LOG.info("Getting Cell id for Location {}, {}, {}", lat, lon, cellId);

			KeyFactory keyFactory = getDataStoreInstance().newKeyFactory().setKind(MatchmakerProp.datastoreKind);
			Key key = keyFactory.newKey(cellId);

			Entity currentEntity = findBykey(key);

			if (currentEntity != null) {

				allocation = findAllocation(currentEntity);
				return allocation;

			} else {
				// call allocate for new server and add new entity
				allocation = allocationServiceClient.getAllocation(cellId.toString());
				if (allocation != null && key != null) {
					createOrUpdateCellId(key, allocation.getStatus().getAddress(),
							allocation.getStatus().getPorts().get(0).getPort());

				}

			}

		}

		return allocation;

	}

	private static String getCellId(double latDegrees, double lngDegrees) {
		S2CellId id = S2CellId.fromLatLng(S2LatLng.fromDegrees(latDegrees, lngDegrees));
		return Long.toString(id.parent(10).id());

	}

	List<Value<String>> convertToValueList(List<Value<String>> values, String value) {

		List<Value<String>> result = new ArrayList<Value<String>>();
		result.addAll(values);
		result.add(StringValue.of(value));
		return result;
	}

	List<Value<String>> convertToValueList(String value) {
		List<Value<String>> result = new ArrayList<Value<String>>();
		result.add(StringValue.of(value));
		return result;
	}

	private Datastore getDataStoreInstance() {
		Datastore datastore = DatastoreOptions.getDefaultInstance().getService();
		return datastore;
	}

	private Entity findBykey(Key key) {

		return getDataStoreInstance().get(key);
	}

	private void createOrUpdateCellId(Key key, String ip, String port) {

		Entity entity = findBykey(key);
		// update
		if (entity != null) {

			List<Value<String>> ips = new ArrayList<Value<String>>();
			List<Value<String>> ports = new ArrayList<Value<String>>();
			ips = entity.getList("IP");
			ports = entity.getList("PORT");

			entity = Entity.newBuilder(key).set("IP", convertToValueList(ips, ip))
					.set("PORT", convertToValueList(ports, port)).build();
			getDataStoreInstance().update(entity);
			LOG.info("Successfully updated cell id entity with entity key {}", key);

		}
		// create
		else {

			entity = Entity.newBuilder(key).set("IP", convertToValueList(ip)).set("PORT", convertToValueList(port))
					.build();
			getDataStoreInstance().put(entity);
			LOG.info("Successfully created new  cell id entity with entity key {}", key);

		}

	}

	@SuppressWarnings("resource")
	private int getPlayerCount(String ip, String port) throws IOException {

		int gsPort = Integer.parseInt(port);
		int playerCount = 0;
		String msg;

		DatagramSocket socket = new DatagramSocket();
		InetAddress address = InetAddress.getByName(ip);
		byte[] buf = "PCOUNT".getBytes();
		DatagramPacket packet = new DatagramPacket(buf, buf.length, address, gsPort);
		socket.send(packet);

		buf = new byte[256];
		packet = new DatagramPacket(buf, buf.length);

		// blocks until a packet is received
		socket.receive(packet);
		msg = new String(packet.getData()).trim();

		LOG.info("Message from " + packet.getAddress().getHostAddress() + ": PlayerCount: " + msg);

		playerCount = Integer.parseInt(msg);

		return playerCount;

	}

	private Allocation allocationBuilder(String key, String ip, String port) {

		Allocation allocation = new Allocation();
		Status status = new Status();
		status.setAddress(ip);
		status.setCellId(key);
		ArrayList<Ports> portList = new ArrayList<>();
		Ports p = new Ports();
		p.setPort(port);
		portList.add(p);
		status.setPorts(portList);
		allocation.setStatus(status);
		return allocation;
	}

	private Allocation findAllocation(Entity current) throws IOException {

		Iterator<Value<?>> ipIterator = current.getList("IP").iterator();
		Iterator<Value<?>> portIterator = current.getList("PORT").iterator();
		String cellId = current.getKey().getName();
		boolean found = false;
		Allocation allocation = null;

		while (ipIterator.hasNext() && portIterator.hasNext()) {

			String ip = ipIterator.next().get().toString();
			String port = portIterator.next().get().toString();
			LOG.info("Looking to get number of players from {}, {}", ip, port);
			if (getPlayerCount(ip, port) < MAX_PLAYER_ALLOWED) {
				found = true;
				allocation = allocationBuilder(cellId, ip, port);
				break;
			}
		}

		if (!found) {

			// add a new allocated server in the key update entity
			allocation = allocationServiceClient.getAllocation(cellId);
			if (allocation != null) {
				createOrUpdateCellId(current.getKey(), allocation.getStatus().getAddress(),
						allocation.getStatus().getPorts().get(0).getPort());
			}

		}

		return allocation;

	}
}
