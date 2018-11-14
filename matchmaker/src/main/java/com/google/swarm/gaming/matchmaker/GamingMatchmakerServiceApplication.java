package com.google.swarm.gaming.matchmaker;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.http.client.support.BasicAuthorizationInterceptor;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestTemplate;

@SpringBootApplication
public class GamingMatchmakerServiceApplication {
	public static final Logger LOG = LoggerFactory.getLogger(GamingMatchmakerServiceApplication.class);

	public static void main(String[] args) {
		SpringApplication.run(GamingMatchmakerServiceApplication.class, args);
	
	}

	@Bean
	RestTemplate loadBalancedRestTemplate() {
		final RestTemplate restTemplate = new RestTemplate();
		restTemplate.getInterceptors()
				.add(new BasicAuthorizationInterceptor("v1GameClientKey", "EAEC945C371B2EC361DE399C2F11E"));
		return restTemplate;

	}

	@Component
	public static class MatchmakerProp {

		public static int maxNumberofUsers;
		public static String allocationServiceURL;
		public static String datastoreKind;

		@Autowired
		public MatchmakerProp(@Value("${max.users}") String maxNumberofUsers,
				@Value("${allocation.service.url}") String allocationServiceURL, 
				@Value("${data_store_kind}") String datastoreKind ) {
			MatchmakerProp.maxNumberofUsers = Integer.parseInt(maxNumberofUsers);
			MatchmakerProp.allocationServiceURL = allocationServiceURL;
			MatchmakerProp.datastoreKind= datastoreKind;
			LOG.info("*****Allocation Service URL:{} *****Max Number of Users: {} **** Data Store Kind: {}", 
					MatchmakerProp.allocationServiceURL,
					MatchmakerProp.maxNumberofUsers,
					MatchmakerProp.datastoreKind);
		}
	}

}
