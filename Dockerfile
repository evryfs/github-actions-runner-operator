FROM quay.io/quarkus/centos-quarkus-maven:20.0.0-java11 as builder
COPY pom.xml /project/
COPY src /project/src
RUN mvn clean package

FROM ubuntu:latest
WORKDIR /app
COPY --from=builder /project/target/com.tietoevry.fss.garo.mainkt .
ENTRYPOINT ["./com.tietoevry.fss.garo.mainkt"]
