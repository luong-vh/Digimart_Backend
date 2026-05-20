# SE357-ShoppingManagement-BE

<p align="center">
  <a href="https://www.uit.edu.vn/" title="Trường Đại học Công nghệ Thông tin" style="border: 5;">
    <img src="https://i.imgur.com/WmMnSRt.png" alt="Trường Đại học Công nghệ Thông tin | University of Information Technology">
  </a>
</p>

## Members

<a name="thanhvien"></a>
| STT | ID | Full name | Github | Email |
| ------ |:-------------:|:----------------------:|:----------------------------------------------------:|:-----------------------:|
| 1 | 23520905 | Võ Hồng Lương |[luong-vh](https://github.com/luong-vh) |23520905@gm.uit.edu.vn |
| 2 | 23520905 | Võ Hồng Lương |[luong-vh](https://github.com/luong-vh) |23520905@gm.uit.edu.vn |

## How to run

### If you have Go on your machine

Ensure you have Redis running on port 6379. For local `go run .`, keep
`REDIS_ADDR=localhost:6379` in `.env` and start Redis separately:

```powershell
docker compose up -d redis
```

Build the project with `go build -o <your app name>`.

Then you can run the built executable.

## Or running with Docker

First, build Docker image with `docker build -t shop-be .`

Then, you can run with Docker Compose `docker compose up`. In Docker Compose,
the app service overrides `REDIS_ADDR` to `redis:6379` so it can reach the Redis
service through the Compose network.
