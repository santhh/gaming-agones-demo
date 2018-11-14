package com.google.swarm.gaming.matchmaker.model;

import java.util.ArrayList;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonProperty;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@NoArgsConstructor
public class Allocation {

	@JsonProperty("status")
	private Status status;

	@Data
	@NoArgsConstructor
	public static class Status {

		private String cellId;

		@JsonIgnore
		private String state;
		@JsonProperty("address")
		private String address;

		@JsonIgnore
		private String nodeName;
		@JsonProperty("ports")

		private ArrayList<Ports> ports;

		@Data
		@NoArgsConstructor
		@AllArgsConstructor
		public static class Ports {

			@JsonIgnore
			private String name;

			@JsonProperty("port")
			private String port;

		}
	}

}
