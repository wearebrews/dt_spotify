from pyngrok import ngrok
import os

url: str = ngrok.connect(port=8080)
url = url.replace("http", "https", 1)

os.environ["BASE_URL"] = url
os.environ["SPOTIFY_CLIENT_ID"] = ""
os.environ["SPOTIFY_CLIENT_SECRET"] = "" 

os.system("./main")


