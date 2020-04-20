FROM quay.io/quarkus/centos-quarkus-maven:20.0.0-java11 as builder
COPY pom.xml /project/
COPY src /project/src
RUN mvn clean package

FROM quay.io/evryfs/base-java:java11
WORKDIR /app
COPY --from=builder /project/target/garo-runner.jar .
COPY --from=builder /project/target/lib lib/
ENTRYPOINT ["java", "-jar", "garo-runner.jar"]
