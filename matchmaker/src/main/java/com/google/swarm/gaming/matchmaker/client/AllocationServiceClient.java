package com.google.swarm.gaming.matchmaker.client;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestTemplate;

import com.google.swarm.gaming.matchmaker.GamingMatchmakerServiceApplication.MatchmakerProp;
import com.google.swarm.gaming.matchmaker.model.Allocation;

import lombok.RequiredArgsConstructor;

@Component
@RequiredArgsConstructor
public class AllocationServiceClient {
	static private final Logger LOG = LoggerFactory.getLogger(AllocationServiceClient.class);

	private final RestTemplate restTemplate;

	public Allocation getAllocation(String cellId) {

		HttpHeaders httpHeaders = new HttpHeaders();
		httpHeaders.set("Accept", MediaType.APPLICATION_JSON_VALUE);
		HttpEntity<?> httpEntity = new HttpEntity<>(httpHeaders);

		ResponseEntity<Allocation> response = restTemplate.exchange(MatchmakerProp.allocationServiceURL, HttpMethod.GET,
				httpEntity, Allocation.class);

		response.getBody().getStatus().setCellId(cellId);
		LOG.info("Allocation Service Response Received: " + response.getBody());
		return response.getBody();

	}

}
